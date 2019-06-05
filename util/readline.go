package util

import (
	"errors"
	"strings"

	"github.com/chzyer/readline"
)

func Ask(prompt string) (string, error) {
	if !IsTTY {
		return "", errors.New("No interactive terminal detected. Try running rain in non-interactive mode.")
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt: prompt + " ",
	})
	if err != nil {
		return "", err
	}

	answer, err := rl.Readline()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(answer), nil
}

func Confirm(defaultYes bool, prompt string) (bool, error) {
	extra := " (y/N)"

	if defaultYes {
		extra = " (Y/n)"
	}

	answer, err := Ask(prompt + extra)
	if err != nil {
		return false, err
	}

	if answer == "Y" || (defaultYes && answer == "") {
		return true, nil
	}

	return false, nil
}
