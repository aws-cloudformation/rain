package cmd

func init() {
	Commands["validate"] = Command{
		Type: TEMPLATE,
		Help: "Validate templates",
		Run:  validateCommand,
	}
}

func validateCommand(args []string) {
	panic("Not implemented")
}
