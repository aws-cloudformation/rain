package deploy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListToMapOneString(t *testing.T) {
	parameterList := []string{"aoeu=htns"}
	expectedParameterMap := map[string]string{
		"aoeu": "htns",
	}

	assert.Equal(t,
		expectedParameterMap,
		listToMap("", parameterList),
	)
}

func TestListToMapCommaValue(t *testing.T) {
	parameterList := []string{"aoeu=htns,thing"}
	expectedParameterMap := map[string]string{
		"aoeu": "htns,thing",
	}

	assert.Equal(t,
		expectedParameterMap,
		listToMap("", parameterList),
	)
}

func TestListToMapMultipleEntriesValue(t *testing.T) {
	parameterList := []string{"aoeu=htns", "key2=value2", "key3=YetAnotherValue"}
	expectedParameterMap := map[string]string{
		"aoeu": "htns",
		"key2": "value2",
		"key3": "YetAnotherValue",
	}

	assert.Equal(t,
		expectedParameterMap,
		listToMap("", parameterList),
	)
}

