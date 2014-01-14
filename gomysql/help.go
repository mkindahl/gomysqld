package main

import (
	"fmt"
	"mysqld/cmd"
	"os"
	"strings"
)

var helpCmd = cmd.Command{
	Brief: "Give help on commands and groups",

	Description: `Show the help text for the command or group
	under the words given. If there are any extreneous words, an
	error message will be given instead and the help not
	printed.

        If no arguments are provided at all, this help message is
        shown.`,

	Synopsis: "WORD ...",
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		// If no arguments were given, we show help on "help"
		if len(args) == 0 {
			args = []string{"help"}
		}

		// Locate command
		_, node, rest := ctx.Locate(args)
		if len(rest) > 0 {
			cmds := strings.Join(rest, " ")
			return fmt.Errorf("Extreneous arguments: %q", cmds)
		}

		node.PrintHelp(os.Stdout)
		return nil
	},
}

func init() {
	context.RegisterCommand([]string{"help"}, &helpCmd)
}
