package main

import (
	"fmt"
	"mysqld/cmd"
	"mysqld/stable"
)

var initCmd = cmd.Command{
	Brief: "Initialize the MySQL Server stable",

	Description: `This command will create an empty stable in the
        location where distributions and server can be added.

        It will also try to find an existing installation and add it
        as a "synthetic distribution" so that you can create servers
        based on what you have installed on your machine.

        This can be useful if you, for example, want to test that some
        application work the same way on your currently installed
        servers and some other version.`,

	Synopsis:   "LOCATION",
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

		// Look for an existing mysqld installation at some
		// known places. Note that the files might not be
		// located all in the same directory, so we have to
		// build a structure for the distribution to match
		// what is expected from an added distribution.

		return err
	},
}

func init() {
	context.RegisterCommand([]string{"init"}, &initCmd)
}
