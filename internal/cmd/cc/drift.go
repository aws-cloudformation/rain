package cc

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
	"github.com/aws-cloudformation/rain/internal/aws/ccapi"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func runDrift(cmd *cobra.Command, args []string) {

	name := args[0]

	if !Experimental {
		panic("Please add the --experimental arg to use this feature")
	}

	spinner.Push("Downloading state file")

	bucketName := s3.RainBucket(false)

	key := fmt.Sprintf("%v/%v.yaml", STATE_DIR, name) // deployments/name

	obj, err := s3.GetObject(bucketName, key)
	if err != nil {
		panic(fmt.Errorf("unable to download state: %v", err))
	}

	config.Debugf("State file: %s", obj)

	template, err := parse.String(string(obj))
	if err != nil {
		panic(err)
	}

	spinner.Pop()

	if err := runDriftOnState(name, template, bucketName, key); err != nil {
		panic(err)
	}
}

func runDriftOnState(name string, template cft.Template, bucketName string, key string) error {

	resources, err := template.GetSection(cft.Resources)
	if err != nil {
		return err
	}

	_, err = template.GetSection(cft.State)
	if err != nil {
		return err
	}

	// Display deployment meta-data

	fmt.Println()
	fmt.Println("Checking for drift on existing deployment")
	fmt.Println()
	fmt.Print(console.Blue("Deployment name:  "))
	fmt.Print(console.Cyan(fmt.Sprintf("%s\n", name)))

	fmt.Print(console.Blue("State file:       "))
	fmt.Print(console.Cyan(fmt.Sprintf("s3://%s/%s\n", bucketName, key)))

	localPath, err := template.GetNode(cft.State, "FilePath")
	if err != nil {
		panic(err)
	}
	fmt.Print(console.Blue("Local path:       "))
	fmt.Print(console.Cyan(fmt.Sprintf("%s\n", localPath.Value)))

	lastWrite, err := template.GetNode(cft.State, "LastWriteTime")
	if err != nil {
		panic(err)
	}
	fmt.Print(console.Blue("Last write time:  "))
	fmt.Print(console.Cyan(fmt.Sprintf("%s\n", lastWrite.Value)))

	resourceModels, err := template.GetNode(cft.State, "ResourceModels")
	if err != nil {
		panic(err)
	}

	fmt.Println()

	selections := make([]selection, 0)

	// Query each resource and stop to ask how to handle drift after each one
	for i := 0; i < len(resources.Content); i += 2 {
		resourceName := resources.Content[i].Value
		resourceNode := resources.Content[i+1]
		_, resourceModel := s11n.GetMapValue(resourceModels, resourceName)
		if resourceModel == nil {
			panic(fmt.Errorf("expected %s to have a ResourceModel", resourceName))
		}

		selection, err := handleDrift(resourceName, resourceNode, resourceModel)
		if err != nil {
			panic(err)
		}
		selections = append(selections, selection)
	}

	// Check to see if the user elected to change anything
	hasChanges := false
	for _, selection := range selections {
		if selection.Action != doNothing {
			hasChanges = true
			break
		}
	}

	// Summarize all changes that will be made and ask the user to confirm
	if !hasChanges {
		fmt.Println("No changes were made to your infrastructure or to the state file.")
		return nil
	}

	fmt.Println("The following changes will be made:")
	fmt.Println()
	for _, selection := range selections {
		switch selection.Action {
		case changeLiveState:
			fmt.Println("   ⚡ Change Live State for", selection.ResourceName)
		case changeStateFile:
			fmt.Println("   📄 Change state file for", selection.ResourceName)
		}
	}
	fmt.Println()

	// Confirm and then actually make the changes
	if !console.Confirm(true, "Do you wish to continue?") {
		fmt.Println("Deployment cancelled. No changes have been made to the state file or to live state")
		return nil
	}

	// Set the global template reference for resolving intrinsics
	deployedTemplate = template

	hasStateFileChanges := false
	for _, selection := range selections {
		switch selection.Action {
		case changeLiveState:
			spinner.Push(fmt.Sprintf("   ⚡ Changing Live State for %s", selection.ResourceName))

			// Download the schema
			schema, err := cfn.GetTypeSchema(selection.ResourceType)
			if err != nil {
				console.Errorf("unable to load schema for %s: %v", selection.ResourceName, err)
				break
			}

			// Look at the schema to get read only props and remove them
			var schemaMap map[string]any
			json.Unmarshal([]byte(schema), &schemaMap)

			roProps := make([]string, 0)

			readOnly, exists := schemaMap["readOnlyProperties"]
			if exists {
				config.Debugf("readOnly: %v", readOnly)
				for _, p := range readOnly.([]any) {
					roProps = append(roProps, strings.Replace(p.(string), "/properties/", "", 1))
				}
			}

			// Resolve intrinsics
			resolvedNode, err := Resolve(selection.DeploymentResource)
			if err != nil {
				console.Errorf("Unable to resolve %s: %v", selection.ResourceName, err)
				break
			}

			newPriorMap := make(map[string]any)
			for k, v := range selection.LiveModel {
				if !slices.Contains(roProps, k) {
					newPriorMap[k] = v
				}
			}

			priorJson, _ := json.Marshal(newPriorMap)

			model, err := ccapi.UpdateResource(selection.ResourceName,
				selection.ResourceIdentifier, resolvedNode, string(priorJson))
			if err != nil {
				msg := "unable to update live state for %s: %v"
				console.Errorf(msg, selection.ResourceName, err)
				break
			}
			config.Debugf("Updated %s, got model: %s", selection.ResourceName, model)

			spinner.Pop()

			fmt.Println(console.Green(fmt.Sprintf("Updated %s", selection.ResourceName)))

		case changeStateFile:
			hasStateFileChanges = true

			spinner.Push(fmt.Sprintf("   📄 Changing state file for %s", selection.ResourceName))

			_, resourceModel := s11n.GetMapValue(resourceModels, selection.ResourceName)
			config.Debugf("About to change state ResoureModel for %s: %v", selection.ResourceName, selection.LiveModel)
			var replacementNode yaml.Node
			replacementNode.Encode(selection.LiveModel)
			node.SetMapValue(resourceModel, "Model", &replacementNode)

			spinner.Pop()
		}
	}

	if hasStateFileChanges {
		lastWrite.Value = time.Now().Format(time.RFC3339)
		str := format.String(template, format.Options{JSON: false, Unsorted: false})
		err = s3.PutObject(bucketName, key, []byte(str))
		if err != nil {
			console.Errorf("unable to write updated state file to bucket: %v", err)
		} else {
			fmt.Println("State file updated successfully")
		}
	}
	return nil
}

type action int

const (
	changeLiveState action = 1
	changeStateFile action = 2
	doNothing       action = 3
)

type selection struct {
	ResourceName       string
	Action             action
	Text               string
	LiveModel          map[string]any
	StateModel         map[string]any
	ResourceIdentifier string
	ResourceNode       *yaml.Node
	ResourceType       string
	DeploymentResource *Resource
}

func handleDrift(resourceName string, resourceNode *yaml.Node, model *yaml.Node) (selection, error) {

	retval := selection{ResourceName: resourceName, Action: doNothing}

	_, t := s11n.GetMapValue(resourceNode, "Type")
	if t == nil {
		return retval, fmt.Errorf("resource %s expected to have Type", resourceName)
	}
	_, id := s11n.GetMapValue(model, "Identifier")
	if id == nil {
		return retval, fmt.Errorf("resource model %s expected to have Identifier", resourceName)
	}
	title := fmt.Sprintf("%s (%s %s)", resourceName, t.Value, id.Value)

	spinner.Push(fmt.Sprintf("Querying CCAPI: %s", title))

	liveModelJson, err := ccapi.GetResource(id.Value, t.Value)
	if err != nil {
		return retval, err
	}
	spinner.Pop()

	_, stateModel := s11n.GetMapValue(model, "Model")
	if stateModel == nil {
		return retval, fmt.Errorf("expected State %s to have Model", resourceName)
	}

	var liveModelMap map[string]any
	err = json.Unmarshal([]byte(liveModelJson), &liveModelMap)
	if err != nil {
		return retval, err
	}

	var modelMap map[string]any
	err = stateModel.Decode(&modelMap)
	if err != nil {
		panic(err)
	}

	stateModelJsonb, _ := json.Marshal(modelMap)
	stateModelJson := string(stateModelJsonb)

	// In order to resolve intrinsics, we need to store the resources
	// in the global resMap as *Resource pointers
	r := &Resource{
		Name:       resourceName,
		Type:       t.Value,
		Node:       resourceNode,
		Identifier: id.Value,
		Model:      stateModelJson,
		PriorJson:  liveModelJson,
	}

	retval.DeploymentResource = r

	// Also store a reference in the global map for later if we
	// need to resolve intrinsics
	resMap[resourceName] = r

	d := diff.CompareMaps(modelMap, liveModelMap)

	liveIcon := "⚡"
	storedIcon := "📄"
	checkIcon := "✅"
	resourceIcon := "🔎 "

	// Is there any reason not to show the icons?
	//
	// if console.NoColour {
	// 	liveIcon = "*"
	// 	storedIcon = "."
	// 	checkIcon = "> "
	// 	resourceIcon = "-> "
	// }

	if d.Mode() == diff.Unchanged {
		fmt.Println(console.Green(resourceIcon + title + "... Ok!"))
	} else {
		fmt.Println(console.Red(resourceIcon + title + "... Drift detected!"))
		fmt.Println()

		// Show a diff of the live state and stored state
		fmt.Println("    ========== " + liveIcon + " Live state " + liveIcon + " ==========")
		fmt.Println("   ", colorDiff(d.Format(true)))
		reverse := diff.CompareMaps(liveModelMap, modelMap)
		fmt.Println("    ========== " + storedIcon + " Stored state " + storedIcon + " ==========")
		fmt.Println("   ", colorDiff(reverse.Format(true)))

		// Ask the user that they want to do

		selections := []selection{
			{Action: changeLiveState, Text: "Change the live state so it matches the state file (changes your infrastructure!)"},
			{Action: changeStateFile, Text: "Change the state file so that it matches live state"},
			{Action: doNothing, Text: "Do nothing"},
		}

		activeFormat := " {{ .Text | magenta }}"
		selectedFormat := " {{ .Text | blue }}"

		if console.NoColour {
			activeFormat = " {{ .Text }}"
			selectedFormat = " {{ .Text }}"
		}

		prompt := promptui.Select{
			Label: fmt.Sprintf("What would you like to do with %s?", resourceName),
			Items: selections,
			Templates: &promptui.SelectTemplates{
				Label:    "{{ . }}",
				Active:   checkIcon + activeFormat,
				Inactive: "   {{ .Text }}",
				Selected: checkIcon + selectedFormat,
			},
		}

		idx, _, err := prompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return retval, err
		}

		retval.Action = selections[idx].Action
		retval.LiveModel = liveModelMap
		retval.StateModel = modelMap
		retval.ResourceIdentifier = id.Value
		retval.ResourceNode = resourceNode
		retval.ResourceType = t.Value

	}

	fmt.Println()
	return retval, nil
}

// colorDiff hacks the diff output to colorize it
func colorDiff(s string) string {
	lines := strings.Split(s, "\n")
	f := "%s "
	unchanged := fmt.Sprintf(f, diff.Unchanged)
	ret := make([]string, 0)
	for _, line := range lines {
		// Lines look like these:
		// (=) QueryDefinitionId: 0abf4544-b551-4b79-93d0-6f7f294cdbaa
		// (>) QueryString: fields @message, @timestamp
		tokens := strings.SplitAfterN(line, " ", 2)
		if len(tokens) != 2 {
			ret = append(ret, console.Yellow(line)) // Shouldn't happen
		} else {
			if tokens[0] == unchanged {
				ret = append(ret, console.Green(tokens[1]))
			} else {
				if console.NoColour {
					ret = append(ret, "! "+tokens[1])
				} else {
					ret = append(ret, console.Red(tokens[1]))
				}
			}
		}
	}
	retval := strings.Join(ret, "\n    ")
	if console.NoColour {
		// Offset the ! so it stands out and the props are still aligned
		retval = strings.Replace(retval, "    ! ", "  ! ", -1)
	}
	return retval
}

var CCDriftCmd = &cobra.Command{
	Use:   "drift <name>",
	Short: "Compare the state file to the live state of the resources",
	Long: `When deploying templates with the cc command, a state file is created and stored in the rain assets bucket. This command outputs a diff of that file and the actual state of the resources, according to Cloud Control API. You can then apply the changes by changing the live state, or by modifying the state file.
`,
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run:                   runDrift,
}

func init() {
	addCommonParams(CCDriftCmd)
}
