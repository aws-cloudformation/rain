package text

// Code generated. DO NOT EDIT.

// Bold returns a Text struct that wraps the supplied text in ANSI colour
func Bold(text string) Text {
	return Text{
		text:   text,
		colour: "\x1b[1;37m",
	}
}

// Green returns a Text struct that wraps the supplied text in ANSI colour
func Green(text string) Text {
	return Text{
		text:   text,
		colour: "\x1b[0;32m",
	}
}

// Grey returns a Text struct that wraps the supplied text in ANSI colour
func Grey(text string) Text {
	return Text{
		text:   text,
		colour: "\x1b[1;30m",
	}
}

// Orange returns a Text struct that wraps the supplied text in ANSI colour
func Orange(text string) Text {
	return Text{
		text:   text,
		colour: "\x1b[0;33m",
	}
}

// Red returns a Text struct that wraps the supplied text in ANSI colour
func Red(text string) Text {
	return Text{
		text:   text,
		colour: "\x1b[1;31m",
	}
}

// White returns a Text struct that wraps the supplied text in ANSI colour
func White(text string) Text {
	return Text{
		text:   text,
		colour: "\x1b[1;37m",
	}
}

// Yellow returns a Text struct that wraps the supplied text in ANSI colour
func Yellow(text string) Text {
	return Text{
		text:   text,
		colour: "\x1b[1;33m",
	}
}

// Plain returns a Text struct that always returns the supplied text, unformatted
func Plain(text string) Text {
	return Text{
		text:   text,
		colour: "",
	}
}
