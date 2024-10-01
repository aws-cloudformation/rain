package deploy

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/aws/cfn"
	"github.com/aws-cloudformation/rain/internal/aws/cloudfront"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/console"
	"github.com/aws-cloudformation/rain/internal/console/spinner"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"gopkg.in/yaml.v3"
)

func addCommandArgs(run *yaml.Node, cmd *exec.Cmd, isBefore bool, stackName string) error {
	_, runArgs, _ := s11n.GetMapValue(run, "Args")
	var outputValues []types.Output
	gotOutputs := false
	var err error
	if runArgs != nil && runArgs.Kind == yaml.SequenceNode {
		for _, arg := range runArgs.Content {
			tokens := strings.Split(arg.Value, " ")
			config.Debugf("Args tokens (%d): %v", len(tokens), tokens)
			if len(tokens) == 2 && tokens[0] == "Rain::OutputValue" {
				if isBefore {
					return errors.New("Rain::OutputValue is invalid for RunBefore Args")
				}
				// Go get the stack and get the output value to use as the arg
				if !gotOutputs {
					outputValues, err = cfn.GetStackOutputs(stackName)
					if err != nil {
						return err
					}
					gotOutputs = true
				}
				foundOutput := false
				for _, output := range outputValues {
					config.Debugf("output %+v", output)
					if *output.OutputKey == tokens[1] {
						foundOutput = true
						cmd.Args = append(cmd.Args, *output.OutputValue)
						break
					}
				}
				if !foundOutput {
					return fmt.Errorf("did not find output value %s", tokens[1])
				}
			} else {
				// Otherwise assume that the argument is literal
				config.Debugf("Literal Arg: %s", arg.Value)
				cmd.Args = append(cmd.Args, arg.Value)
			}
		}
	}
	return nil
}

// processMetadataBefore looks for Rain commands in resource Metadata
// that need to be run before deployment.
// For CREATE and UPDATE operations, a Run node on a bucket
// will run a script, which should run before deployment in case there
// are any errors. Then after deployment, the Content node is processed
func processMetadataBefore(template cft.Template, stackName string, rootDir string) error {

	// For some reason Package created an extra document node
	// (And CreateChangeSet is ok with this...?)
	template.Node = template.Node.Content[0]

	buckets := template.GetResourcesOfType("AWS::S3::Bucket")
	for _, bucket := range buckets {
		_, n, _ := s11n.GetMapValue(bucket.Node, "Metadata")
		if n == nil {
			continue
		}
		_, n, _ = s11n.GetMapValue(n, "Rain")
		if n == nil {
			continue
		}

		// Run a script before deployment
		err := Run(n, "RunBefore", stackName, rootDir)
		if err != nil {
			return err
		}
	}

	return nil

}

// Run checks to ses if either RunBefore or RunAfter is defined in the
// Rain metadata sections and runs and external command, like a build script.
// Args to the command can be literal strings or stack output lookups,
// if this is "RunAfter".
//
// Example:
//
//	Metadata:
//	  Rain:
//	    Content: site/dist
//	    RunBefore:
//	      Command: buildsite.sh
//	    RunAfter: buildsite.sh
//	      Command: buildsite.sh
//	      Args:
//	        - Rain::OutputValue  RestApiInvokeURL
//	        - Rain::OutputValue  RedirectURI
//	        - Rain::OutputValue  AppName
//	        - Rain::OutputValue  AppClientId
func Run(n *yaml.Node, key string, stackName string, rootDir string) error {

	if key != "RunBefore" && key != "RunAfter" {
		return errors.New("key must be RunBefore or RunAfter")
	}

	_, run, _ := s11n.GetMapValue(n, key)
	if run == nil {
		config.Debugf("%s not found", key)
		return nil
	}

	_, runCmd, _ := s11n.GetMapValue(run, "Command")
	if runCmd == nil {
		config.Debugf("Run missing Command")
		return nil
	}
	commandToRun := runCmd.Value
	config.Debugf("Running %s", commandToRun)

	relativePath := filepath.Join(".", rootDir, commandToRun)
	absPath, absErr := filepath.Abs(relativePath)
	if absErr != nil {
		config.Debugf("filepath.Abs failed? %s", absErr)
		return absErr
	}
	cmd := exec.Command(absPath)
	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = rootDir
	err := addCommandArgs(run, cmd, (key == "RunBefore"), stackName)
	if err != nil {
		return err
	}
	config.Debugf("Run command args: %v", cmd.Args)
	err = cmd.Run()
	if err != nil {
		fmt.Println(console.Red(fmt.Sprintf(
			"Run %s failed with %s: %s",
			commandToRun, err, stderr.String())))
		fmt.Println(stdout.String())
		return err
	}
	fmt.Printf("Successfully ran %s\n", commandToRun)
	return nil
}

// processMetadataAfter looks for Rain commands in resource Metadata
// that need to be run after deployment.
// For CREATE and UPDATE operations, a Content node on a bucket
// will upload local assets to the bucket.
func processMetadataAfter(template cft.Template, stackName string, rootDir string) error {

	// For some reason Package created an extra document node
	// (And CreateChangeSet is ok with this...?)
	template.Node = template.Node.Content[0]

	buckets := template.GetResourcesOfType("AWS::S3::Bucket")
	for _, bucket := range buckets {
		logicalId := bucket.LogicalId
		_, n, _ := s11n.GetMapValue(bucket.Node, "Metadata")
		if n == nil {
			continue
		}
		_, n, _ = s11n.GetMapValue(n, "Rain")
		if n == nil {
			continue
		}

		// Run a script after deployment but before uploading the content
		err := Run(n, "RunAfter", stackName, rootDir)
		if err != nil {
			return err
		}

		_, contentPath, _ := s11n.GetMapValue(n, "Content")
		if contentPath == nil {
			continue
		}

		// Ignore RAIN_NO_CONTENT or an empty string
		if contentPath.Value == "" || contentPath.Value == "RAIN_NO_CONTENT" {
			continue
		}

		// Assume contentPath.Value is a directory and Put all files
		p := filepath.Join(rootDir, contentPath.Value)

		// Get the bucket name
		sr, err := cfn.GetStackResource(stackName, logicalId)
		if err != nil {
			return err
		}
		bucketName := *sr.PhysicalResourceId

		spinner.Push(fmt.Sprintf("Uploading the contents of %s to %s", p, bucketName))

		// Recursively walk the directory and upload all files
		err = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				config.Debugf("error walking %s: %v", p, err)
				return err
			}
			if !info.IsDir() {
				f, readErr := os.ReadFile(path)
				if readErr != nil {
					config.Debugf("error reading %s: %v", path, err)
					return readErr
				}
				// Get rid of the local directory path
				// For example, if the local file is myfiles/foo/bar.txt,
				// put bar.txt into the bucket
				putPath := strings.Replace(path, p, "", 1)
				putPath = strings.TrimLeft(putPath, "/")
				putErr := s3.PutObject(bucketName, putPath, f)
				config.Debugf("PutObject: %s/%s", bucketName, putPath)
				if putErr != nil {
					config.Debugf("error putting %s/%s: %v", bucketName, putPath, putErr)
					return putErr
				}
			}
			return nil
		})
		spinner.Pop()
		if err != nil {
			return err
		}
		fmt.Printf("Copied contents of %s to %s\n", p, bucketName)

		// Look for a LogicalId of a Distribution to invalidate
		_, dlid, _ := s11n.GetMapValue(n, "DistributionLogicalId")
		if dlid != nil {

			spinner.Push(fmt.Sprintf("Invalidating CloudFront Distribution %s", dlid.Value))

			// Look up the distribution id
			sr, err := cfn.GetStackResource(stackName, dlid.Value)
			if err != nil {
				return err
			}
			did := *sr.PhysicalResourceId
			config.Debugf("About to invalidate %s", did)

			// Invalidate the distribution
			err = cloudfront.Invalidate(did)
			spinner.Pop()
			if err != nil {
				return err
			}
			fmt.Printf("CloudFront Distribution %s invalidated\n", did)
		}
	}

	return nil

}
