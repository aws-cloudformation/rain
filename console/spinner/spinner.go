// Package spinner contains functions for displaying progress updates
// with a spinning icon that shows the user that progress is being made.
package spinner

import (
	"fmt"
	"time"

	"github.com/aws-cloudformation/rain/console"
)

var spin = []string{
	`-`, `\`, `|`, `/`,
}

var spinRunning = false
var spinHasTimer = false

var spinStatus = ""
var spinCount = 0
var spinStartTime time.Time

func init() {
	if console.IsTTY {
		go func() {
			for {
				if spinRunning {
					spinUpdate()
					spinCount = (spinCount + 1) % len(spin)
				}

				time.Sleep(time.Second / 2)
			}
		}()
	}
}

func spinUpdate() {
	console.ClearLine()

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

// Status enables the spinner and displays the provided message
func Status(status string) {
	spinRunning = true
	spinStatus = status
}

// Timer enables the spinner and displays a timer counting upwards from 0
func Timer() {
	spinRunning = true
	spinStartTime = time.Now()
	spinHasTimer = true
	spinStatus = ""
}

// Stop disables the spinner
func Stop() {
	spinHasTimer = false
	spinRunning = false
	spinStatus = ""
	console.ClearLine()
}

// Update redisplays the spinner. Use this if your programme has updated
// the screen and may have interfered with the spinners display location.
func Update() {
	if console.IsTTY {
		spinUpdate()
	}
}
