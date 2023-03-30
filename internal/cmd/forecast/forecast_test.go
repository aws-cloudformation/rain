package forecast

import "testing"

func TestForecastMethods(t *testing.T) {
	forecast := makeForecast("A::B::C", "Id")
	forecast.Add(true, "Succeeded")
	forecast.Add(false, "Failed")

	if forecast.GetNumChecked() != 2 {
		t.Errorf("Expected 2 checks")
	}

	if forecast.GetNumFailed() != 1 {
		t.Errorf("Expected 1 failure")
	}

	if forecast.GetNumPassed() != 1 {
		t.Errorf("Expected 1 pass")
	}

	f2 := makeForecast("D::E::F", "Id2")
	f2.Add(false, "f2 fail")
	forecast.Append(f2)

	if forecast.GetNumChecked() != 3 || forecast.GetNumFailed() != 2 {
		t.Errorf("Append did not append")
	}
}
