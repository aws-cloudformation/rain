package cmd

// Disabled for now
/*
func init() {
	// Find plugins
	path := os.Getenv("PATH")

	for _, dir := range strings.Split(path, ":") {
		bins, err := filepath.Glob(dir + "/cfn-*")
		if err != nil {
			panic(err)
		}

		for i, _ := range bins {
			bin := bins[i]

			name := string(bin[len(dir)+5:])

			cmd := &cobra.Command{
				Use:   name,
				Short: fmt.Sprintf("Executes the external command '%s'", bin[len(dir)+1:]),
				Run: func(cmd *cobra.Command, args []string) {
					util.RunAttached(bin, args...)
				},
			}

			cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
				util.RunAttached(bin, "--help")
			})

			Rain.AddCommand(cmd)
		}
	}
}
*/
