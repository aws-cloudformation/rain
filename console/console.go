// Package console contains utility functions for working with text consoles
package console

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/andrew-d/go-termutil"
	"github.com/chzyer/readline"
)

// IsTTY will be true if stdout is connected to a true terminal
var IsTTY bool

// HasColour is true if your program is running a platform that supports ANSI colours
var HasColour bool

func init() {
	IsTTY = termutil.Isatty(os.Stdout.Fd())
	HasColour = runtime.GOOS != "windows"
}

// Clear removes all text from the console and puts the cursor in the top-left corner
func Clear(content string) {
	if IsTTY && HasColour {
		fmt.Print("\033[1;1H\033[2J")
	} else {
		fmt.Println()
	}

	fmt.Println(content)
}

// ClearLine removes all text from the current line and puts the cursor on the left
func ClearLine() {
	if IsTTY && HasColour {
		fmt.Print("\033[1G\033[2K")
	} else {
		fmt.Println()
	}
}

// Ask prints the supplied prompt and then waits for user input which is returned as a string.
func Ask(prompt string) string {
	if !IsTTY {
		panic(errors.New("No interactive terminal detected. Try running rain in non-interactive mode"))
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt: prompt + " ",
	})
	if err != nil {
		panic(fmt.Errorf("Unable to get user input: %s", err))
	}

	answer, err := rl.Readline()
	if err != nil {
		panic(fmt.Errorf("Unable to get user input: %s", err))
	}

	return strings.TrimSpace(answer)
}

// Confirm asks the user for "y" or "n" and returns true if the response was "y".
// defaultYes is used to determine whether (y/N) or (Y/n) is displayed after the prompt.
func Confirm(defaultYes bool, prompt string) bool {
	extra := " (y/N)"

	if defaultYes {
		extra = " (Y/n)"
	}

	answer := Ask(prompt + extra)

	if strings.ToUpper(answer) == "Y" || (defaultYes && answer == "") {
		return true
	}

	return false
}
