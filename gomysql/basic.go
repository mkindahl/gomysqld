package main

import (
	"flag"
	"fmt"
	"mysqld/stable"
)

var flagRoot string

var initCmd = Command{
	brief:    "Initialize the MySQL Server stable",
	synopsis: "LOCATION",
	description: `This command will create an empty stable in the location where
distributions and server can be added.`,
	skipStable: true,
	body: func(ctx *Context, args []string) error {
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

var listGrp = Group{
	brief: "Commands for listing things",
	description: `This group contain commands for listing all kinds of things regarding
the stable.`,
}

var addGrp = Group{
	brief: "Commands for adding things",
}

var removeGrp = Group{
	brief: "Commands for removing things",
}

func init() {
	flag.StringVar(&flagRoot, "root", ".", "Root directory for stable")
	context.RegisterGroup([]string{"list"}, &listGrp)
	context.RegisterGroup([]string{"add"}, &addGrp)

	context.RegisterCommand([]string{"init"}, &initCmd)
}
