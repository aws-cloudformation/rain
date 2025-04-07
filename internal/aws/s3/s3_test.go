package s3

import (
	"strings"
	"testing"
)

func TestParseURI(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		wantBucket  string
		wantKey     string
		wantErr     bool
		errContains string
	}{
		{
			name:       "Valid URI with bucket and key",
			uri:        "s3://mybucket/path/to/object.txt",
			wantBucket: "mybucket",
			wantKey:    "path/to/object.txt",
			wantErr:    false,
		},
		{
			name:       "Valid URI with bucket only",
			uri:        "s3://mybucket",
			wantBucket: "mybucket",
			wantKey:    "",
			wantErr:    false,
		},
		{
			name:        "Empty URI",
			uri:         "",
			wantBucket:  "",
			wantKey:     "",
			wantErr:     true,
			errContains: "invalid s3 uri",
		},
		{
			name:       "URI with empty key",
			uri:        "s3://mybucket/",
			wantBucket: "mybucket",
			wantKey:    "",
			wantErr:    false,
		},
		{
			name:       "URI with complex key",
			uri:        "s3://mybucket/folder1/folder2/file.txt",
			wantBucket: "mybucket",
			wantKey:    "folder1/folder2/file.txt",
			wantErr:    false,
		},
		{
			name:       "URI with special characters in key",
			uri:        "s3://mybucket/path with spaces/file-name_123.txt",
			wantBucket: "mybucket",
			wantKey:    "path with spaces/file-name_123.txt",
			wantErr:    false,
		},
		{
			name:        "URI without s3:// prefix",
			uri:        "mybucket/path/to/object.txt",
			wantBucket: "",
			wantKey:    "",
			wantErr:    true,
			errContains: "must start with s3://",
		},
		{
			name:        "URI with invalid bucket name (too short)",
			uri:        "s3://ab",
			wantBucket: "",
			wantKey:    "",
			wantErr:    true,
			errContains: "length must be between 3 and 63",
		},
		{
			name:        "URI with invalid bucket name (too long)",
			uri:        "s3://" + strings.Repeat("a", 64),
			wantBucket: "",
			wantKey:    "",
			wantErr:    true,
			errContains: "length must be between 3 and 63",
		},
		{
			name:        "URI with key exceeding length limit",
			uri:        "s3://mybucket/" + strings.Repeat("a", 1025),
			wantBucket: "",
			wantKey:    "",
			wantErr:    true,
			errContains: "length exceeds 1024 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBucket, gotKey, err := ParseURI(tt.uri)

			// Check error cases
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseURI() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ParseURI() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			// Check non-error cases
			if err != nil {
				t.Errorf("ParseURI() unexpected error = %v", err)
				return
			}

			if gotBucket != tt.wantBucket {
				t.Errorf("ParseURI() gotBucket = %v, want %v", gotBucket, tt.wantBucket)
			}

			if gotKey != tt.wantKey {
				t.Errorf("ParseURI() gotKey = %v, want %v", gotKey, tt.wantKey)
			}
		})
	}
}
