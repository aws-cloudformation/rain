package util

func Bold(text string) Text {
	return Text{
		text:   text,
		colour: "\x1b[1;37m",
	}
}

func Orange(text string) Text {
	return Text{
		text:   text,
		colour: "\x1b[0;33m",
	}
}

func Yellow(text string) Text {
	return Text{
		text:   text,
		colour: "\x1b[1;33m",
	}
}

func Red(text string) Text {
	return Text{
		text:   text,
		colour: "\x1b[1;31m",
	}
}

func Green(text string) Text {
	return Text{
		text:   text,
		colour: "\x1b[0;32m",
	}
}

func Grey(text string) Text {
	return Text{
		text:   text,
		colour: "\x1b[0;37m",
	}
}

func White(text string) Text {
	return Text{
		text:   text,
		colour: "\x1b[0;30m",
	}
}
