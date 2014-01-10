package main

import (
	"fmt"
)

var addDistCmd = Command{
	brief:    "Add a distribution to the stable",
	synopsis: "add distribution PATH",
	body: func(ctx *Context, args []string) error {
		_, err := ctx.Stable.AddDist(args[0])
		return err
	},
}

var listDistCmd = Command{
	brief:    "List information about distributions",
	synopsis: "list distributions",
	body: func(ctx *Context, args []string) error {
		for _, dist := range ctx.Stable.Distro {
			fmt.Printf("%s\n", dist.Name)
			fmt.Printf("\t      Version: %s\n", dist.Version)
			fmt.Printf("\tServerVersion: %s\n", dist.ServerVersion)
		}
		return nil
	},
}

var removeDistCmt = Command{
	brief:    "Remove a distribution from the stable",
	synopsis: "NAME",
	body: func(ctx *Context, args []string) error {
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

func init() {
	context.RegisterCommand([]string{"add", "distribution"}, &addDistCmd)
	context.RegisterCommand([]string{"list", "distributions"}, &listDistCmd)
}
