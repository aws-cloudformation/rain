package deploy

import (
	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws-cloudformation/rain/internal/node"
	"github.com/aws-cloudformation/rain/internal/s11n"
)

func deployContent(template cft.Template) error {
	config.Debugf("deployContent about to check for buckets with Rain Content")

	config.Debugf("deployContent template Node: \n%s", node.ToSJson(template.Node))

	// For some reason Package created an extra document node
	// (And CreateChangeSet is ok with this...?)
	template.Node = template.Node.Content[0]

	// Iterate over resources looking for buckets
	resources, err := template.GetSection(cft.Resources)
	if err != nil {
		return err
	}

	for i := 0; i < len(resources.Content); i += 2 {
		logicalId := resources.Content[i].Value
		bucket := resources.Content[i+1]
		_, typ, _ := s11n.GetMapValue(bucket, "Type")
		if typ == nil {
			continue
		}
		if typ.Value != "AWS::S3::Bucket" {
			continue
		}

		config.Debugf("deployContent bucket: %s \n%v", logicalId, node.ToSJson(bucket))
		_, n, _ := s11n.GetMapValue(bucket, "Metadata")
		if n == nil {
			continue
		}
		config.Debugf("deployContent found Metadata")
		_, n, _ = s11n.GetMapValue(n, "Rain")
		if n == nil {
			continue
		}
		_, contentPath, _ := s11n.GetMapValue(n, "Content")
		if contentPath == nil {
			continue
		}
		config.Debugf("deployContent found contentPath for resource: %s",
			contentPath.Value)

		// TODO: We need to know the bucket name!
		// We might not have it if it wasn't named in the template
		// Use CCAPI to get the name based on the logical id

	}

	return nil

}
