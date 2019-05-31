package util

import (
	"errors"
	"strings"

	"github.com/chzyer/readline"
)

func Ask(prompt string) string {
	if !IsTTY {
		Die(errors.New("No interactive terminal detected. Try running rain in non-interactive mode."))
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt: prompt + " ",
	})
	if err != nil {
		Die(err)
	}

	answer, err := rl.Readline()
	if err != nil {
		Die(err)
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
