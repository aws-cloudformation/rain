// Package console contains utility functions for working with text consoles
package console

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/gookit/color"
	"github.com/nathan-fiscaletti/consolesize-go"
	"golang.org/x/term"
)

// IsTTY will be true if stdout is connected to a true terminal
var IsTTY bool

// isANSI will be true if console supports ANSI escape code. It is for Windows only.
var isANSI bool

// NoColour should be false if you want output to be coloured
var NoColour = false

func init() {
	IsTTY = term.IsTerminal(int(os.Stdout.Fd()))
	isANSI = true
}

// Size returns the width and height of the console in characters
func Size() (int, int) {
	return consolesize.GetConsoleSize()
}

// CountLines returns the number of lines that would be taken up by the given string
func CountLines(input string) int {
	input = color.ClearCode(input)

	if input == "" {
		return 0
	}

	w, _ := Size()

	if w == 0 {
		return 0
	}

	count := 0
	for _, line := range strings.Split(input, "\n") {
		d := int(math.Ceil(float64(len([]rune(line))) / float64(w)))
		if d == 0 {
			d = 1
		}
		count += d
	}

	return count
}

// ClearLine removes all text from the current line and puts the cursor on the left
func ClearLine() {
	if IsTTY && isANSI {
		fmt.Print("\033[G\033[K")
	} else {
		fmt.Println()
	}
}

// ClearLines removes all text from the previous n lines (starting with the current line) and puts the cursor on the left
func ClearLines(n int) {
	if !IsTTY {
		return
	}

	for i := 0; i < n; i++ {
		ClearLine()
		if i < n-1 {
			if IsTTY && isANSI {
				fmt.Print("\033[F")
			}
		}
	}
}

// Ask prints the supplied prompt and then waits for user input which is returned as a string.
func Ask(prompt string) string {
	if !IsTTY {
		panic(errors.New("no interactive terminal detected; try running rain in interactive mode (e.g. without --yes)"))
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt: prompt + " ",
	})
	if err != nil {
		panic(fmt.Errorf("unable to get user input: %w", err))
	}

	answer, err := rl.Readline()
	if err != nil {
		panic(fmt.Errorf("unable to get user input: %w", err))
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
