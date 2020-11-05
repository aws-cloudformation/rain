package console

import (
	"fmt"
	"github.com/gookit/color"
)

func wrap(c color.Color) func(...interface{}) string {
	return func(in ...interface{}) string {
		if NoColour || !IsTTY {
			return fmt.Sprint(in...)
		}

		return c.Render(in...)
	}
}

// Blue returns the input as a string of blue-coloured text if the console supports colours
var Blue = wrap(color.Blue)

// Green returns the input as a string of green-coloured text if the console supports colours
var Green = wrap(color.Green)

// Grey returns the input as a string of grey-coloured text if the console supports colours
var Grey = wrap(color.Gray)

// Red returns the input as a string of red-coloured text if the console supports colours
var Red = wrap(color.LightRed)

// White returns the input as a string of white-coloured text if the console supports colours
var White = wrap(color.White)

// Yellow returns the input as a string of yellow-coloured text if the console supports colours
var Yellow = wrap(color.Yellow)

// Plain returns the input as a string of normal-coloured text if the console supports colours
var Plain = wrap(color.Normal)
