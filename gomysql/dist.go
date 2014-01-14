package main

import (
	"fmt"
	"mysqld/cmd"
	"os"
	"text/tabwriter"
)

var addDistCmd = cmd.Command{
	Brief: "Add a distribution to the stable",

	Description: `A distribution will be added to the stable using an
	archive of a binary distribution. Either a tar file (gzipped or not), a
	zip file, or an unpacked binary distribution can be used. If a directory
	is given, a symlink will be created that point to the directory.`,

	Synopsis: "add distribution PATH",
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		_, err := ctx.Stable.AddDist(args[0])
		return err
	},

	Init: func(cmd *cmd.Command) {
		cmd.Flags.String("name", "",
			"Name of distribution, if different from directory name")
	},
}

var showDistCmd = cmd.Command{
	Brief: "Show information about distributions",

	Description: `Show all the added distributions with both the version and
	the server version. The 'SERVER VERSION' is fetched from calling 'mysqld
	--version', and the 'VERSION' is retrieved from the include file. In
	some cases, different builds can produce a server version that contain
	extra information, but the version is the base version of the server,
	regardless of build options.`,

	Synopsis: "show distributions",
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		tw := tabwriter.NewWriter(os.Stdout, 8, 0, 2, ' ', tabwriter.AlignRight)
		fmt.Fprintf(tw, "%s\t%s\t%s\t\n", "NAME", "VERSION", "SERVER VERSION")
		for _, dist := range ctx.Stable.Distro {
			fmt.Fprintf(tw,
				"%s\t%s\t%s\t\n",
				dist.Name, dist.Version, dist.ServerVersion)
		}
		tw.Flush()
		return nil
	},
}

var removeDistCmt = cmd.Command{
	Brief: "Remove a distribution from the stable",

	Description: `The distribution will be completely removed from
	the stable, including all servers that are based on that
	distribution.`,

	Synopsis: "NAME",
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		// Remove the distribution
		return ctx.Stable.DelDistByName(args[0])
	},
}

var distGrp = cmd.Group{
	Brief:       "Commands for working with distributions",
	Description: `All commands for working with distributions are in this group. `,
}

func init() {
	context.RegisterGroup([]string{"distribution"}, &distGrp)
	context.RegisterCommand([]string{"distribution", "add"}, &addDistCmd)
	context.RegisterCommand([]string{"distribution", "show"}, &showDistCmd)
}
