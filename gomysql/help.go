package main

import (
	"fmt"
	"mysqld/cmd"
	"os"
	"strings"
)

var helpCmd = cmd.Command{
	Brief:       "Give help on commands and groups",
	Synopsis:    "WORD ...",
	Description: `Show the help text for the command or group under the words given. If there are any extreneous words, an error message will be given instead and the help not printed.`,
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		// If no arguments were given, we show help on "help"
		if len(args) == 0 {
			args = []string{"help"}
		}

		// Locate command
		_, node, rest := ctx.Locate(args)
		if len(rest) > 0 {
			return fmt.Errorf("Extreneous arguments to help command: %q", strings.Join(rest, " "))
		}

		node.PrintHelp(os.Stdout)
		return nil
	},
}

func init() {
	context.RegisterCommand([]string{"help"}, &helpCmd)
}
