package cmd

func init() {
	Commands["rm"] = Command{
		Type: STACK,
		Help: "Delete stacks",
		Run:  rmCommand,
	}
}

func rmCommand(args []string) {
	panic("Not implemented")
}
