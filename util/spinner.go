package util

import (
	"fmt"
	"time"
)

var spin = []string{
	`-`, `\`, `|`, `/`,
}

var spinRunning = false
var spinHasTimer = false

var spinStatus = ""
var spinCount = 0
var spinStartTime time.Time

func spinUpdate() {
	if spinRunning {
		ClearLine()

		if spinHasTimer {
			fmt.Printf("%s %s %s",
				spin[spinCount],
				time.Now().Sub(spinStartTime).Truncate(time.Second),
				spinStatus,
			)
		} else {
			fmt.Printf("%s %s", spin[spinCount], spinStatus)
		}
	}

}

func startSpinner() {
	if IsTTY {
		go func() {
			for {
				spinUpdate()
				spinCount = (spinCount + 1) % len(spin)
				time.Sleep(time.Second / 2)
			}
		}()
	}
}

func SpinStatus(status string) {
	spinRunning = true
	spinStatus = status
}

func SpinStartTimer() {
	spinRunning = true
	spinStartTime = time.Now()
	spinHasTimer = true
	spinStatus = ""
}

func SpinStop() {
	spinHasTimer = false
	spinRunning = false
	spinStatus = ""
	ClearLine()
}
