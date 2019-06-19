package util

import (
	"fmt"
	"time"
)

var spin = []string{
	`-`, `\`, `|`, `/`,
}

var spinRunning = false
var spinStatus = ""

func startSpinner() {
	if IsTTY {
		go func() {
			count := 0

			for {
				if spinRunning {
					ClearLine()
					fmt.Printf("%s %s", spin[count], spinStatus)
					count = (count + 1) % len(spin)
				}

				time.Sleep(time.Second / 2)
			}
		}()
	}
}

func SpinStatus(status string) {
	spinStatus = status
	spinRunning = true
}

func SpinStop() {
	ClearLine()
	spinRunning = false
}
