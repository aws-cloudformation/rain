//go:build windows

// Package console contains utility functions for working with text consoles
package console

import (
	"os"

	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

func init() {
	var fd = os.Stdout.Fd()
	IsTTY = term.IsTerminal(int(fd))
	isANSI = isANSISupport(fd)
}

func isANSISupport(fd uintptr) bool {
	var consoleHandle = windows.Handle(fd)
	var consoleMode uint32
	err := windows.GetConsoleMode(consoleHandle, &consoleMode)
	if err != nil {
		return false
	}
	if (consoleMode & windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) != windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING {
		return false
	}
	return true
}
