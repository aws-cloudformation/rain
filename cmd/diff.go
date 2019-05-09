package cmd

func init() {
	Commands["diff"] = Command{
		Type: TEMPLATE,
		Run:  diffCommand,
		Help: "Compare templates with other templates or stacks",
	}
}

func diffCommand(args []string) {
	panic("Not implemented")
}
