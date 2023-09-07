package forecast

import "testing"

func TestResourceEstimate(t *testing.T) {
	resourceName := "AWS::ACMPCA::Certificate"
	action := Create
	est, err := GetResourceEstimate(resourceName, action)
	if err != nil {
		t.Error(err)
		return
	}
	if est != 1 {
		t.Errorf("expected AWS::ACMPCA::Certificate create to return 1")
	}
}
