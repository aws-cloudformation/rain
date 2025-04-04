package pkg

import (
	"testing"
)

func TestGetArrayIndexFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		{
			name:     "Basic index",
			input:    "Content[1].Arn",
			expected: 1,
			hasError: false,
		},
		{
			name:     "Zero index",
			input:    "Resources[0]",
			expected: 0,
			hasError: false,
		},
		{
			name:     "Large index",
			input:    "MyArray[42].Value",
			expected: 42,
			hasError: false,
		},
		{
			name:     "No brackets",
			input:    "NoIndexHere",
			expected: 0,
			hasError: true,
		},
		{
			name:     "Invalid index",
			input:    "Invalid[x]",
			expected: 0,
			hasError: true,
		},
		{
			name:     "Multiple indices - should return first",
			input:    "Multiple[1][2]",
			expected: 1,
			hasError: false,
		},
		{
			name:     "Empty brackets",
			input:    "Empty[]",
			expected: 0,
			hasError: true,
		},
		{
			name:     "Missing closing bracket",
			input:    "Missing[1",
			expected: 0,
			hasError: true,
		},
		{
			name:     "Negative index",
			input:    "Negative[-1]",
			expected: -1,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetArrayIndexFromString(tt.input)
			
			// Check error status
			if (err != nil) != tt.hasError {
				t.Errorf("GetArrayIndexFromString(%q) error = %v, wantErr %v", 
					tt.input, err, tt.hasError)
				return
			}
			
			// If we expect no error, check the result value
			if !tt.hasError && result != tt.expected {
				t.Errorf("GetArrayIndexFromString(%q) = %d, want %d", 
					tt.input, result, tt.expected)
			}
		})
	}
}
