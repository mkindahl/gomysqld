// Copyright (c) 2014, Oracle and/or its affiliates. All rights reserved.

// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; version 2 of the License.

// This program is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program; if not, write to the Free Software
// Foundation, Inc., 51 Franklin St, Fifth Floor, Boston, MA 02110-1301
// USA

package main

import (
	"fmt"
	"mysqld/cmd"
	"mysqld/stable"
	"os"
)

var version = "0.1.0"

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

var versionCmd = cmd.Command{
	Brief: "Show tool version",

	Description: "This command will show the version of the tool.",
	SkipStable:  true,
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		fmt.Printf("%s version %s\n", os.Args[0], version)
		return nil
	},
}

func init() {
	context.RegisterCommand([]string{"init"}, &initCmd)
	context.RegisterCommand([]string{"version"}, &versionCmd)
}
