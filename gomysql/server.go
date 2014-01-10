package main

import (
	"fmt"
	"mysqld/cmd"
	"mysqld/stable"
	"os"
	"strings"
	"text/tabwriter"
)

var srvGrp = cmd.Group{
	Brief: "Manipulating server instances",

	Description: `All commands for manipulating and working with
server instances are in this group. `,
}

var addServerCmd = cmd.Command{
	Brief:    "Add a server to the stable",
	Synopsis: "NAME",

	Description: `This command will create a new server using a
previously added distribution and add it to the stable. If a value 
to -dist is given, the distributions having the provided string as
substring will be used. If less than or more than one distribution
matching, an error will be returned. The default for the distribution
is the empty string, which will pick every available distribution,
which is convenient if you have only one distribution.`,

	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("Wrong number of arguments")
		}

		candidates := []*stable.Dist{}
		flag := cmd.Flags.Lookup("dist")
		for key, dist := range ctx.Stable.Distro {
			if strings.Contains(key, flag.Value.String()) {
				candidates = append(candidates, dist)
			}
		}
		if len(candidates) == 0 {
			return fmt.Errorf("No distribution containing %q", flag.Value.String())
		} else if len(candidates) > 1 {
			return fmt.Errorf("Ambigous choice.")
		}
		dist := candidates[0]

		if _, err := ctx.Stable.AddServer(args[0], dist); err != nil {
			return fmt.Errorf("Unable to create server %s: %s", args[0], err.Error())
		}
		return nil
	},

	Init: func(cmd *cmd.Command) {
		cmd.Flags.String("dist", "", "Distribution to create the server from.")
	},
}

var removeServerCmd = cmd.Command{
	Brief:    "Remove a server from the stable",
	Synopsis: "NAME",
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		return ctx.Stable.DelServerByName(args[0])
	},
}

var listServersCmd = cmd.Command{
	Brief: "List servers in the stable",
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		if len(args) > 0 {
			argStr := strings.Join(args, " ")
			return fmt.Errorf("Wrong number of arguments %q", argStr)
		}
		tw := tabwriter.NewWriter(os.Stdout, 8, 0, 2, ' ', tabwriter.AlignRight)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t\n", "NAME", "HOST", "PORT", "VERSION")
		for _, srv := range ctx.Stable.Server {
			fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t\n", srv.Name, srv.Host, srv.Port, srv.Dist.ServerVersion)
		}
		tw.Flush()
		return nil
	},
}

func init() {
	context.RegisterGroup([]string{"server"}, &srvGrp)
	context.RegisterCommand([]string{"server", "add"}, &addServerCmd)
	context.RegisterCommand([]string{"server", "remove"}, &removeServerCmd)
	context.RegisterCommand([]string{"server", "show"}, &listServersCmd)
	context.RegisterCommand([]string{"server", "start"}, &startServersCmd)
	context.RegisterCommand([]string{"server", "stop"}, &stopServersCmd)
}
