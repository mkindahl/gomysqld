package main

import (
	"flag"
	"fmt"
	"mysqld/cmd"
	"mysqld/stable"
)

var flagRoot string

var initCmd = cmd.Command{
	Brief:    "Initialize the MySQL Server stable",
	Synopsis: "LOCATION",
	Description: `This command will create an empty stable in the location where
distributions and server can be added.`,
	SkipStable: true,
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		// Check the number of arguments and provide a reasonable
		// error message
		if len(args) != 1 {
			return fmt.Errorf("command 'init' require LOCATION")
		}

		// This command creates a stable, so we update the stable
		// field in the context to ensure that surrounding code can
		// use it.
		stbl, err := stable.CreateStable(args[0])
		if err == nil {
			ctx.Stable = stbl
		}
		return err
	},
}

var addGrp = cmd.Group{
	Brief: "Commands for adding things",
}

var removeGrp = cmd.Group{
	Brief: "Commands for removing things",
}

func init() {
	flag.StringVar(&flagRoot, "root", ".", "Root directory for stable")

	context.RegisterCommand([]string{"init"}, &initCmd)
}
