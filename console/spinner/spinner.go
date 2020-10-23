// Package spinner contains functions for displaying progress updates
// with a spinning icon that shows the user that progress is being made.
package spinner

import (
	"fmt"
	"time"

	"github.com/aws-cloudformation/rain/console"
	"github.com/aws-cloudformation/rain/console/text"
)

var spin = []string{"˙", "·", ".", " "}
var drops = 3

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

				time.Sleep(time.Second / 7)
			}
		}()
	}
}

func spinUpdate() {
	console.ClearLine()

	if spinHasTimer {
		fmt.Printf("%s%s%s %s %s",
			text.Blue(spin[spinCount]),
			text.Blue(spin[(spinCount+3)%len(spin)]),
			text.Blue(spin[(spinCount+5)%len(spin)]),
			time.Now().Sub(spinStartTime).Truncate(time.Second),
			spinStatus,
		)
	} else {
		fmt.Printf("%s %s%s%s",
			spinStatus,
			text.Blue(spin[spinCount]),
			text.Blue(spin[(spinCount+3)%len(spin)]),
			text.Blue(spin[(spinCount+5)%len(spin)]),
		)
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

// Pause stops the spinner until Resume is called
func Pause() {
	spinRunning = false
}

// Resume causes the spinner to restart following a call to Pause
func Resume() {
	spinRunning = true
}

// Update redisplays the spinner. Use this if your programme has updated
// the screen and may have interfered with the spinners display location.
func Update() {
	if console.IsTTY {
		spinUpdate()
	}
}
