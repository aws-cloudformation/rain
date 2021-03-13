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

func TestRepairValuesWithCommasOneBrokenParameter(t *testing.T) {
	brokenCommaValue := []string{"key=value1", "value2"}
	expectedRepairedValue := []string{"key=value1,value2"}

	repairedValue, _ := repairValuesWithCommas(brokenCommaValue)
	assert.Equal(t,
		expectedRepairedValue,
		repairedValue,
	)
}

func TestRepairValuesWithCommasOneBrokenParameterAndOneCorrect(t *testing.T) {
	brokenCommaValue := []string{"key=value1", "value2", "key2=anotherValue"}
	expectedRepairedValue := []string{"key=value1,value2", "key2=anotherValue"}

	repairedValue, _ := repairValuesWithCommas(brokenCommaValue)
	assert.Equal(t,
		expectedRepairedValue,
		repairedValue,
	)
}

func TestRepairValuesWithCommasPanicOnNoKeyStart(t *testing.T) {
	faultyValues := []string{"NoKeyedValue"}

	_, err := repairValuesWithCommas(faultyValues)
	assert.NotNil(t, err)
}

func TestRepairValuesWithCommasReturnsEmptyOnEmptyInput(t *testing.T) {
	repairedValuesEmpty, _ := repairValuesWithCommas([]string{})
	assert.Equal(t, repairedValuesEmpty, []string{})
}
