// Package spinner contains functions for displaying progress updates
// with a spinning icon that shows the user that progress is being made.
package spinner

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws-cloudformation/rain/internal/console"
)

var spin = []string{"˙", "·", ".", " "}
var drops = 3

var hasTimer = false

var statuses []string
var count = 0
var startTime time.Time
var paused = false

var lastLine = ""

func init() {
	statuses = make([]string, 0)

	go func() {
		for console.IsTTY {
			if !paused && len(statuses) > 0 {
				update()
				count = (count + 1) % len(spin)
			}

			time.Sleep(time.Second / 7)
		}
	}()
}

func update() {
	if !console.IsTTY {
		return
	}

	console.ClearLines(console.CountLines(lastLine))

	if !paused && len(statuses) > 0 {
		status := strings.TrimSpace(statuses[len(statuses)-1])

		if hasTimer {
			lastLine = fmt.Sprintf("%s%s%s %s %s",
				console.Blue(spin[count]),
				console.Blue(spin[(count+3)%len(spin)]),
				console.Blue(spin[(count+5)%len(spin)]),
				time.Now().Sub(startTime).Truncate(time.Second),
				status,
			)
		} else {
			lastLine = fmt.Sprintf("%s %s%s%s",
				status,
				console.Blue(spin[count]),
				console.Blue(spin[(count+3)%len(spin)]),
				console.Blue(spin[(count+5)%len(spin)]),
			)
		}

		fmt.Print(lastLine)
	}
}

// Push enables the spinner and displays the provided message
func Push(status string) {
	statuses = append(statuses, status)

	update()
}

// StartTimer enables the spinner and displays a timer counting upwards from 0
func StartTimer(status string) {
	startTime = time.Now()
	hasTimer = true

	Push(status)
}

// StopTimer disables the timer
func StopTimer() {
	hasTimer = false

	Pop()
}

// Pop removes the move recent status and stops the spinner if there are no more messages
func Pop() {
	if len(statuses) > 0 {
		statuses = statuses[:len(statuses)-1]
	}

	if console.IsTTY {
		update()
	}
}

// Pause pauses the spinner so that you can interact with the console
func Pause() {
	paused = true

	if console.IsTTY {
		update()
	}
}

// Resume resumes the spinner
func Resume() {
	paused = false

	if console.IsTTY {
		update()
	}
}

// Stop empties all spinner messages and stops the spinner
func Stop() {
	statuses = make([]string, 0)

	if console.IsTTY {
		update()
	}
}

// Update causes the spinner to update - use this if you have changed the display and need the spinner to redraw
func Update() {
	update()
}
