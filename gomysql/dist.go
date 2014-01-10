package main

import (
	"fmt"
	"mysqld/cmd"
	"os"
	"text/tabwriter"
)

var addDistCmd = cmd.Command{
	Brief:    "Add a distribution to the stable",
	Synopsis: "add distribution PATH",
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		_, err := ctx.Stable.AddDist(args[0])
		return err
	},

	Init: func(cmd *cmd.Command) {
		cmd.Flags.String("name", "", "Name of distribution, if different from directory name")
	},
}

var listDistCmd = cmd.Command{
	Brief:    "List information about distributions",
	Synopsis: "list distributions",
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		tw := tabwriter.NewWriter(os.Stdout, 8, 0, 2, ' ', tabwriter.AlignRight)
		fmt.Fprintf(tw, "%s\t%s\t%s\t\n", "NAME", "VERSION", "SERVER VERSION")
		for _, dist := range ctx.Stable.Distro {
			fmt.Fprintf(tw, "%s\t%s\t%s\t\n", dist.Name, dist.Version, dist.ServerVersion)
		}
		tw.Flush()
		return nil
	},
}

var removeDistCmt = cmd.Command{
	Brief:    "Remove a distribution from the stable",
	Synopsis: "NAME",
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		// Locate the distribution with the given name

		// Remove all servers using the distribution, we
		// should probably prompt for it before doing it.
		for _, srv := range ctx.Stable.Server {
			ctx.Stable.DelServer(srv)
		}

		// Remove the distribution
		return ctx.Stable.DelDist(args[0])
	},
}

var distGrp = cmd.Group{
	Brief:       "Commands for working with distributions",
	Description: `All commands for working with distributions are in this group. `,
}

func init() {
	context.RegisterGroup([]string{"distribution"}, &distGrp)
	context.RegisterCommand([]string{"distribution", "add"}, &addDistCmd)
	context.RegisterCommand([]string{"distribution", "show"}, &listDistCmd)
}
