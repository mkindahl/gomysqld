package main

import (
	"fmt"
	"strings"
	"os"
)

var helpCmd = Command{
	brief:       "Give help on commands",
	synopsis:    "WORD ...",
	description: "Provide basic help on a the command designated by the list of words.",
	body: func(ctx *Context, args []string) error {
		// If no arguments were given, we show help on "help"
		if len(args) == 0 {
			args = []string{"help"}
		}

		// Locate command
		_, node, rest := ctx.tree.Locate(args)
		if len(rest) > 0 {
			return fmt.Errorf("Extreneous arguments to help command: %q", strings.Join(rest, " "))
		}

		fmt.Println(node.Brief())
		// Options
		node.PrintHelp(os.Stdout)
		return nil
	},
}

func init() {
	context.RegisterCommand([]string{"help"}, &helpCmd)
}