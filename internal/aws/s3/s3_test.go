//go:build func_test

package s3

import (
	"fmt"
	"testing"

	"github.com/aws-cloudformation/rain/internal/aws"
)

func TestPassingBucketName(t *testing.T) {
	tests := [][]string {
		{"custom-bucket-name", "custom-bucket-name"},
		{"", fmt.Sprintf("rain-artifacts-1234567890-%s", aws.Config().Region) },
	}
	for _, test := range tests {
		BucketName = test[0]
		output := RainBucket(false)
		if output != test[1] { 
			t.Errorf("incorrect bucket name, expecting %q but got %q", test[1], output)
		}
	}
}

func TestPassingBucketKeyPrefix(t *testing.T) {
	buckets ["random-bucket"] = true
	content := []byte("some content")
	hash := "290f493c44f5d63d06b374d0a5abd292fae38b92cab2fae5efefe1b0e9347f56"
	tests := [][]string {
		{"some-prefix",        fmt.Sprintf ("some-prefix/%s", hash)},
		{"some-prefix/",       fmt.Sprintf ("some-prefix/%s", hash)},
		{"some-prefix/test",   fmt.Sprintf ("some-prefix/test/%s", hash)},
		{"some-prefix/test/",  fmt.Sprintf ("some-prefix/test/%s", hash)},
		{"",                   hash},
	}
	for _, test := range tests {
		BucketKeyPrefix = test[0]
		output, err := Upload("random-bucket", content)
		if err != nil {
			t.Errorf("unexpected error during upload: %v", err)
		}
		if output != test[1] { 
			t.Errorf("incorrect key, expecting %q but got %q", test[1], output)
		}
	}
}
