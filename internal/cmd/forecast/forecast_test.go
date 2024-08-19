package forecast

import (
	"testing"

	fc "github.com/aws-cloudformation/rain/plugins/forecast"
)

func TestForecastMethods(t *testing.T) {
	input := fc.PredictionInput{}
	input.TypeName = "A::B::C"
	input.LogicalId = "Id"
	forecast := fc.MakeForecast(&input)
	forecast.Add("CODE1", true, "Succeeded", 0)
	forecast.Add("CODE2", false, "Failed", 0)

	if forecast.GetNumChecked() != 2 {
		t.Errorf("Expected 2 checks")
	}

	if forecast.GetNumFailed() != 1 {
		t.Errorf("Expected 1 failure")
	}

	if forecast.GetNumPassed() != 1 {
		t.Errorf("Expected 1 pass")
	}

	input2 := fc.PredictionInput{}
	input2.TypeName = "A::B::C"
	input2.LogicalId = "Id"
	f2 := fc.MakeForecast(&input2)
	f2.Add("CODE2", false, "f2 fail", 0)
	forecast.Append(f2)

	if forecast.GetNumChecked() != 3 || forecast.GetNumFailed() != 2 {
		t.Errorf("Append did not append")
	}
}
