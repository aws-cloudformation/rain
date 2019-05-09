package cmd

func init() {
	Commands["deploy"] = Command{
		Type: TEMPLATE,
		Help: "Deploy templates to stacks",
		Run:  deployCommand,
	}
}

func deployCommand(args []string) {
	panic("Not implemented")
}
