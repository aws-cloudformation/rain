package console

import (
	"fmt"

	"github.com/gookit/color"
)

func wrap(c color.Style) func(...interface{}) string {
	return func(in ...interface{}) string {
		if NoColour || !IsTTY {
			return fmt.Sprint(in...)
		}

		return c.Render(in...)
	}
}

// Sprint wraps color.Sprint with logic to ignore colours if the console does not support colour
func Sprint(in ...interface{}) string {
	out := color.Sprint(in...)

	if NoColour || !IsTTY {
		out = color.ClearCode(out)
	}

	return out
}

// Blue returns the input as a string of blue-coloured text if the console supports colours
var Blue = wrap(color.New(color.Blue))

// Cyan returns the input as a string of cyan-coloured text if the console supports colours
var Cyan = wrap(color.New(color.Cyan))

// Green returns the input as a string of green-coloured text if the console supports colours
var Green = wrap(color.New(color.Green))

// Grey returns the input as a string of grey-coloured text if the console supports colours
var Grey = wrap(color.New(color.Gray))

// Red returns the input as a string of red-coloured text if the console supports colours
var Red = wrap(color.New(color.LightRed))

// White returns the input as a string of white-coloured text if the console supports colours
var White = wrap(color.New(color.Normal, color.OpReverse))

// Yellow returns the input as a string of yellow-coloured text if the console supports colours
var Yellow = wrap(color.New(color.Yellow))

// Bold returns the input as a string of bold text if the console supports colours
var Bold = wrap(color.New(color.Bold))

// Plain returns the input as a string of normal-coloured text if the console supports colours
var Plain = wrap(color.New(color.Normal))
