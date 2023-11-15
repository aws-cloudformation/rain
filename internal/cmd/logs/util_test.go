//go:build func_test

package logs

import (
	"log"
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
	reduceLogsToLength(10, &logs)
	if len(logs) != 10 {
		t.Fatalf("expeced 10 logs got: %d", len(logs))
	}
	logsTestTeardown()
}

func TestReducedLogsByDuration(t *testing.T) {
	logsTestSetup()
	defer logsTestTeardown()

	logs, err := getLogs("logsrange-test-mock-stack", "MockResourceId")
	if err != nil {
		t.Fatalf("%s", err)
	}
	if len(logs) < 30 {
		t.Fatalf("expected log length 30, but got: %d", len(logs))
	}
	logDays := 10
	duration := time.Duration(time.Hour * time.Duration(logDays*-24))
	expectedFirstLogTimeStamp := time.Now()
	expectedLastLogTimeStamp := expectedFirstLogTimeStamp.Add(duration).Add(time.Hour * 24) // adding one day because we only want logs created after the the day before 10 days
	reduceLogsByDuration(duration, &logs)

	// anon function to print out all the relavant logs when tests fail
	printLogs := func() {
		t.Log(duration)
		t.Log(time.Now().Add(duration))
		t.Log("input logs:")
		for _, log := range logs {
			t.Log(log.Timestamp)
		}
	}

	if (logs)[0].Timestamp.Sub(expectedFirstLogTimeStamp) > time.Duration(time.Hour) {
		printLogs()
		t.Fatalf("expected %s timestamp on the latest entry but got %s. The difference is: %s", expectedFirstLogTimeStamp, (logs)[0].Timestamp, (logs)[0].Timestamp.Sub(expectedFirstLogTimeStamp))
	}

	if (logs)[len(logs)-1].Timestamp.Sub(expectedLastLogTimeStamp) > time.Duration(time.Hour) {
		printLogs()
		t.Fatalf("expected %s timestamp on the oldest entry but got %s. The difference is: %s", expectedLastLogTimeStamp, (logs)[len(logs)-1].Timestamp, (logs)[len(logs)-1].Timestamp.Sub(expectedLastLogTimeStamp))
	}
}

func TestReducedLogsWithMultipleFlags(t *testing.T) {
	logsTestSetup()
	defer logsTestTeardown()

	logs, err := getLogs("logsrange-test-mock-stack", "MockResourceId")
	if err != nil {
		t.Fatalf("%s", err)
	}

	reduceLogs(5, 1, &logs)
	expectedNumberOfLogs := 5
	if len(logs) != expectedNumberOfLogs {
		log.Fatalf("expected number of logs is %d, but got %d", expectedNumberOfLogs, len(logs))
	}
}
