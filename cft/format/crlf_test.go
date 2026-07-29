package format_test

import (
	"testing"

	"github.com/aws-cloudformation/rain/cft/format"
	"github.com/aws-cloudformation/rain/cft/parse"
)

func TestCRLFComments(t *testing.T) {
	// Issue #479: Windows CRLF line endings cause extra blank lines between comments
	// Build input with CRLF line endings
	input := "AWSTemplateFormatVersion: 2010-09-09\r\n" +
		"\r\n" +
		"# Comment line 1\r\n" +
		"# Comment line 2\r\n" +
		"# Comment line 3\r\n" +
		"\r\n" +
		"Description: Hello World\r\n"

	// Same input with LF line endings for comparison
	inputLF := "AWSTemplateFormatVersion: 2010-09-09\n" +
		"\n" +
		"# Comment line 1\n" +
		"# Comment line 2\n" +
		"# Comment line 3\n" +
		"\n" +
		"Description: Hello World\n"

	tmplCRLF, err := parse.String(input)
	if err != nil {
		t.Fatalf("Failed to parse CRLF input: %v", err)
	}

	tmplLF, err := parse.String(inputLF)
	if err != nil {
		t.Fatalf("Failed to parse LF input: %v", err)
	}

	outputCRLF := format.String(tmplCRLF, format.Options{})
	outputLF := format.String(tmplLF, format.Options{})

	t.Logf("CRLF output:\n---\n%s\n---", outputCRLF)
	t.Logf("LF output:\n---\n%s\n---", outputLF)

	if outputCRLF != outputLF {
		t.Errorf("CRLF and LF outputs differ.\nCRLF output:\n%s\nLF output:\n%s", outputCRLF, outputLF)
	}
}
