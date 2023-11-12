//go:build func_test

package logs

import (
	"testing"
	"time"
)

func logsTestSetup() {
	allLogs = true
}

func logsTestTeardown() {
	allLogs = false
}
func TestReduceLogsByLength(t *testing.T) {
	logsTestSetup()
	logs, err := getLogs("logsrange-test-mock-stack", "MockResourceId")
	if err != nil {
		t.Fatalf("%s", err)
	}
	if len(logs) < 30 {
		t.Fatalf("expected log length 30, but got: %d", len(logs))
	}
	outputLogs := reduceLogsByLength(10, &logs)
	if len(*outputLogs) != 10 {
		t.Fatalf("expeced 10 logs got: %d", len(*outputLogs))
	}
	logsTestTeardown()
}

func TestReducedLogsByDuration(t *testing.T) {
	logsTestSetup()
	logs, err := getLogs("logsrange-test-mock-stack", "MockResourceId")
	if err != nil {
		t.Fatalf("%s", err)
	}
	if len(logs) < 30 {
		t.Fatalf("expected log length 30, but got: %d", len(logs))
	}
	logDays := 10
	duration := time.Hour * time.Duration(logDays*-24)
	simulatedCurrentTime := time.Date(2010, time.September, 8, 0, 0, 0, 0, time.UTC)
	outputLogs := reduceLogsByDuration(duration, simulatedCurrentTime, &logs)

	if len(*outputLogs) != 10 {
		t.Log(duration)
		t.Log(time.Date(2010, time.September, 8, 0, 0, 0, 0, time.UTC).Add(duration))
		t.Log("input logs:")
		for _, log := range logs {
			t.Log(log.Timestamp)
		}
		t.Fatalf("expected 10 log entries after filtering but got %d", len(*outputLogs))
	}
	firstLogTimeStamp := time.Date(2010, time.September, 8, 0, 0, 0, 0, time.UTC)
	lastLogTimeStamp := time.Date(2010, time.August, 30, 0, 0, 0, 0, time.UTC)
	if !(*outputLogs)[0].Timestamp.Equal(firstLogTimeStamp) {
		t.Fatalf("expected %s timestamp on the latest entry but got %s", firstLogTimeStamp, (*outputLogs)[0].Timestamp)
	}
	if !(*outputLogs)[len(*outputLogs)-1].Timestamp.Equal(lastLogTimeStamp) {
		t.Fatalf("expected %s timestamp on the oldest entry but got %s", lastLogTimeStamp, (*outputLogs)[len(*outputLogs)-1].Timestamp)
	}
	logsTestTeardown()
}
