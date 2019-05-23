package util

import (
	"fmt"
	"os"

	"github.com/andrew-d/go-termutil"
)

var isTTY bool

func init() {
	isTTY = termutil.Isatty(os.Stdout.Fd())
}

type Text struct {
	Text   string
	Colour Colour
}

type Colour string

const (
	None   Colour = ""
	Bold   Colour = "\033[1;37m"
	Orange Colour = "\033[0;33m"
	Yellow Colour = "\033[1;33m"
	Red    Colour = "\033[1;31m"
	Green  Colour = "\033[0;32m"
	Grey   Colour = "\033[0;37m"
	White  Colour = "\033[0;30m"
	End    Colour = "\033[0m"
)

func (t Text) String() string {
	if t.Colour == None || !isTTY {
		return t.Text
	}

	return fmt.Sprintf("%s%s%s", t.Colour, t.Text, End)
}

func (t Text) Len() int {
	return len(t.Text)
}
