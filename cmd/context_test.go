package cmd_test

import "mysqld/cmd"

func ExampleContext_RegisterCommand() {
	context := cmd.NewContext(
		"A tree where we can register commands",
		`This is the root of the command tree and is where
                 we put all commands and subgroups`)

	sampleCmd := &cmd.Command{
		Brief: "A command",
		Body: func(*cmd.Context, *cmd.Command, []string) error {
			return nil
		},
	}

	context.RegisterCommand([]string{"init"}, sampleCmd)
}
