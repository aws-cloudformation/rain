package cfn

import (
	"fmt"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/internal/aws/s3"
	"github.com/aws-cloudformation/rain/internal/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/awslabs/smithy-go/ptr"
)

var liveStatuses = []types.StackStatus{
	"CREATE_IN_PROGRESS",
	"CREATE_FAILED",
	"CREATE_COMPLETE",
	"ROLLBACK_IN_PROGRESS",
	"ROLLBACK_FAILED",
	"ROLLBACK_COMPLETE",
	"DELETE_IN_PROGRESS",
	"DELETE_FAILED",
	"UPDATE_IN_PROGRESS",
	"UPDATE_COMPLETE_CLEANUP_IN_PROGRESS",
	"UPDATE_COMPLETE",
	"UPDATE_ROLLBACK_IN_PROGRESS",
	"UPDATE_ROLLBACK_FAILED",
	"UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS",
	"UPDATE_ROLLBACK_COMPLETE",
	"REVIEW_IN_PROGRESS",
}

func makeTags(tags map[string]string) []types.Tag {
	out := make([]types.Tag, 0)

	for key, value := range tags {
		out = append(out, types.Tag{
			Key:   ptr.String(key),
			Value: ptr.String(value),
		})
	}

	return out
}

func checkTemplate(template cft.Template) (string, error) {
	templateBody := format.String(template, format.Options{})

	if len(templateBody) > 460800 {
		return "", fmt.Errorf("Template is too large to deploy")
	}

	if len(templateBody) > 51200 {
		config.Debugf("Template is too large to deploy directly; uploading to S3.")

		bucket := s3.RainBucket()

		key, err := s3.Upload(bucket, templateBody)

		return fmt.Sprintf("http://%s.s3.amazonaws.com/%s", bucket, key), err
	}

	return templateBody, nil
}
