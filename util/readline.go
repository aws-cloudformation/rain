package util

import (
	"errors"
	"fmt"
	"strings"

	"github.com/chzyer/readline"
)

func Ask(prompt string) string {
	if !IsTTY {
		panic(errors.New("No interactive terminal detected. Try running rain in non-interactive mode."))
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

func Confirm(defaultYes bool, prompt string) bool {
	extra := " (y/N)"

	if defaultYes {
		extra = " (Y/n)"
	}

	answer := Ask(prompt + extra)

	if answer == "Y" || (defaultYes && answer == "") {
		return true
	}

	return false
}
