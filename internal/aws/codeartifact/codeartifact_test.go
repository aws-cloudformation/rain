package codeartifact

import (
	"testing"
)

func TestIncrementSemverMinorVersion(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Increment minor version",
			input:    "1.2.3",
			expected: "1.3.0",
			wantErr:  false,
		},
		{
			name:     "Invalid version format",
			input:    "1.2",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Invalid minor version",
			input:    "1.a.3",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := IncrementSemverMinorVersion(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("IncrementSemverMinorVersion(%s) error = %v, wantErr %v", tc.input, err, tc.wantErr)
				return
			}
			if result != tc.expected {
				t.Errorf("IncrementSemverMinorVersion(%s) = %s, expected %s", tc.input, result, tc.expected)
			}
		})
	}
}

func TestSemverIsGreater(t *testing.T) {
	testCases := []struct {
		name     string
		a        string
		b        string
		expected bool
		wantErr  bool
	}{
		{
			name:     "a is greater than b",
			a:        "1.2.3",
			b:        "1.2.2",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "b is greater than a",
			a:        "1.2.3",
			b:        "1.2.4",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "a and b are equal",
			a:        "1.2.3",
			b:        "1.2.3",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "a has invalid version format",
			a:        "1.2",
			b:        "1.2.3",
			expected: false,
			wantErr:  true,
		},
		{
			name:     "b has invalid version format",
			a:        "1.2.3",
			b:        "1.2.a",
			expected: false,
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := SemverIsGreater(tc.a, tc.b)
			if (err != nil) != tc.wantErr {
				t.Errorf("SemverIsGreater(%s, %s) error = %v, wantErr %v", tc.a, tc.b, err, tc.wantErr)
				return
			}
			if result != tc.expected {
				t.Errorf("SemverIsGreater(%s, %s) = %v, expected %v", tc.a, tc.b, result, tc.expected)
			}
		})
	}
}
