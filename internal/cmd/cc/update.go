package cc

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/diff"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/aws/ccapi"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
	"gopkg.in/yaml.v3"
)

// update compares the template with the current state and returns a
// cloned template annotated with operations to perform on each resource
func update(stateTemplate cft.Template, template cft.Template) (cft.Template, error) {

	// Create a diff between the current state and template
	d := diff.New(stateTemplate, template)
	config.Debugf("update diff:\nMode:%v\n%v", d.Mode(), d.Format(true))

	// Each modified resource needs to be tagged with create-update-delete-none,
	// so that deployResource knows which action to take.
	// We don't need a deep diff, only to identify what resources have changed.

	// In the template, write a node to the resource's State
	/*
		   Resources:
			 MyBucket:
				Type: AWS::S3::Bucket
			 State:
				Action: Create or Update or Delete or None
				Identifier: ...
				ResourceModel: (Might need this for drift detection)
				PriorJson: (We need this for ccapi update)

	*/

	// Figure out what we're doing with each resource (create, update, delete, nothing)
	actions := diff.GetResourceActions(d)

	// Iterate through the state resources and check the diff
	stateRootMap := stateTemplate.Node.Content[0]
	_, stateResourceMap := s11n.GetMapValue(stateRootMap, "Resources")
	if stateResourceMap == nil {
		panic("Expected to find a Resources section in the state template")
	}
	_, stateStateMap := s11n.GetMapValue(stateRootMap, "State")
	if stateStateMap == nil {
		panic("Expected to find a State section in the state template")
	}
	_, stateResourceModels := s11n.GetMapValue(stateStateMap, "ResourceModels")
	if stateResourceModels == nil {
		panic("Expected to find State.ResourceModels in the state template")
	}
	identifiers := make(map[string]string, 0)
	models := make(map[string]*yaml.Node, 0)
	for i, v := range stateResourceModels.Content {
		if i%2 == 0 {
			_, identifier := s11n.GetMapValue(stateResourceModels.Content[i+1], "Identifier")
			if identifier != nil {
				identifiers[v.Value] = identifier.Value
			}
			_, model := s11n.GetMapValue(stateResourceModels.Content[i+1], "Model")
			if model != nil {
				models[v.Value] = node.Clone(model)
			}
		}
	}
	config.Debugf("identifiers: %v", identifiers)

	// Make a copy of the template so the caller still has the original as the user wrote it
	newTemplate := cft.Template{}
	newTemplate.Node = node.Clone(template.Node)

	// Get a reference to the resources in the new template
	newRootMap := newTemplate.Node.Content[0]
	_, newResourceMap := s11n.GetMapValue(newRootMap, "Resources")
	if newResourceMap == nil {
		panic("Expected to find a Resources section in the new template")
	}

	stateResources := make(map[string]*yaml.Node, 0)
	newResources := make(map[string]*yaml.Node, 0)
	resourceActionStates := make(map[string]*yaml.Node) // "State" mapping node

	for i, r := range stateResourceMap.Content {
		if i%2 == 0 {
			name := r.Value
			stateResources[name] = stateResourceMap.Content[i+1]
		}
	}
	for i, r := range newResourceMap.Content {
		if i%2 == 0 {
			name := r.Value
			newResources[name] = newResourceMap.Content[i+1]
			resourceActionStates[name] = node.AddMap(newResources[name], "State")
		}
	}

	// Iterate over the diff and add actions to the output file
	for k, v := range actions {
		rmap, ok := resourceActionStates[k]
		if !ok {
			// Anything missing from the new template is a removed resource to be deleted
			// Clone the resource from the state template and re-add it to the
			// new template with Action: Delete
			if v != diff.Delete {
				return stateTemplate, fmt.Errorf("unexpected missing resource %v has Action %v", k, v)
			}
			newResourceMap.Content = append(newResourceMap.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: k})
			cloned := node.Clone(stateResources[k])
			clonedStateMap := node.AddMap(cloned, "State")
			node.Add(clonedStateMap, "Action", string(v))
			// Add the identifier so we know what to delete
			if identifier, ok := identifiers[k]; ok {
				node.Add(clonedStateMap, "Identifier", identifier)
			}
			newResourceMap.Content = append(newResourceMap.Content, cloned)
		} else {
			// Create, Update, None
			node.Add(rmap, "Action", string(v))
			// Add the identifier so we know what to update
			if identifier, ok := identifiers[k]; ok {
				node.Add(rmap, "Identifier", identifier)
			}
			// Add the resource model that represents the current actual state of
			// the resource based on the ccapi GetResource call
			if model, ok := models[k]; ok {
				modelMap := node.AddMap(rmap, "ResourceModel")
				modelMap.Content = model.Content
			}
			// Add PriorJson to represent the prior properties set by the user
			if v == diff.Update {
				priorProps := ccapi.ToJsonProps(stateResources[k])
				config.Debugf("update setting priorProps: %v", priorProps)
				node.Add(rmap, "PriorJson", priorProps)
			}
		}
	}

	/*
		    Drift remediation.

			Several scenarios are possible here, based on various versions of the resource model:

			1. The current template being submitted.
			2. The prior template, as recorded in the state file.
			3. The actual state of the resource.

			For example, the current template has

			MyQueue:
			  Type: AWS::SQS::Queue
			  Properties:
			  	DelaySeconds: 1

			The new template has

			MyQueue:
			  Type: AWS::SQS::Queue
			  Properties:
			  	DelaySeconds: 2

			And the current state is

			Model:
				Arn: arn:aws:sqs:us-east-1:755952356119:ccdeploy-a
				DelaySeconds: 3
				MaximumMessageSize: 262144
				MessageRetentionPeriod: 345600
				QueueName: ccdeploy-a
				QueueUrl: https://sqs.us-east-1.amazonaws.com/755952356119/ccdeploy-a
				ReceiveMessageWaitTimeSeconds: 0
				SqsManagedSseEnabled: true
				VisibilityTimeout: 30

			The message to the user would be:

				Resource MyQueue has drifted from the prior known state
				and does not match the template you are deploying:

				Current actual state:
					DelaySeconds: 3
				Prior recorded state:
					DelaySeconds: 1
				New template desired state:
				    DelaySeconds: 2

				What would you like to do?
				1) Stop the deployment.
				2) Deploy anyway, applying my latest template as the source of truth
				3) Deploy anyway, applying all of my changes except the drifted properties
				??? Any other choices? Does 3 make sense?

				For CICD, we need to be able to specify the choice with args.

			How much do we care about this? Should this be default behavior, or should
			we add a flag like --warn-on-drift?

			This makes our diff generation more complicated, since there are two
			different diffs to consider.


	*/

	return newTemplate, nil
}

// summarizeChanges prints out a summary of the changes that will be made
// when the template is deployed. This function expects the State property
// to be populated on each resource.
func summarizeChanges(changes cft.Template) {

	d := format.String(changes, format.Options{
		JSON:     false,
		Unsorted: false,
	})
	config.Debugf("change template: %v", d)

	rootMap := changes.Node.Content[0]
	_, resourceMap := s11n.GetMapValue(rootMap, "Resources")
	if resourceMap == nil {
		panic("expected Resources")
	}
	fmt.Println("Summary of deployment changes:")
	for i, v := range resourceMap.Content {
		if i%2 == 0 {
			var action string
			var t string
			var ident string
			name := v.Value
			_, stateMap := s11n.GetMapValue(resourceMap.Content[i+1], "State")
			if stateMap == nil {
				panic(fmt.Sprintf("expected State on resource %v", name))
			}
			_, typeNode := s11n.GetMapValue(resourceMap.Content[i+1], "Type")
			if typeNode == nil {
				panic(fmt.Sprintf("expected Type on resource %v", name))
			}
			t = typeNode.Value
			for si, sv := range stateMap.Content {
				if si%2 == 0 {
					val := stateMap.Content[si+1].Value
					if sv.Value == "Action" {
						action = val
					} else if sv.Value == "Identifier" {
						ident = val
					}
				}
			}
			fmt.Printf("%v\t%v\t%v\t%v\n", name, t, action, ident)
		}
	}

}
