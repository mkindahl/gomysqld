package main

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"strings"
)

var addServerCmd = Command{
	brief: "Add a server to the stable",
	synopsis: "NAME DIST",

	description: `This command will create a new server using a
previously added distribution and add it to the stable.`,

	body: func(ctx *Context, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("Wrong number of arguments")
		}

		dist, ok := ctx.Stable.Distro[args[1]]
		if !ok {
			return fmt.Errorf("No such distribution: %s", args[1])
		}

		if _, err := ctx.Stable.AddServer(args[0], dist); err != nil {
			return fmt.Errorf("Unable to create server %s: %s", args[0], err.Error())
		}
		return nil
	},
}

var removeServerCmd = Command{
	brief: "Remove a server from the stable",
	synopsis: "NAME",
	body: func(ctx *Context, args []string) error {
		return fmt.Errorf("Not Yet Implemented!")
	},
}

var listServersCmd = Command{
	brief: "List servers in the stable",
	body: func(ctx *Context, args []string) error {
		if len(args) > 0 {
			argStr := strings.Join(args, " ")
			return fmt.Errorf("Wrong number of arguments %q", argStr)
		}
		tw := tabwriter.NewWriter(os.Stdout, 8, 0, 2, ' ', tabwriter.AlignRight)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t\n", "Name", "Host", "Port", "Version")
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t\n", "----", "----", "----", "-------")
		for _, srv := range ctx.Stable.Server {
			log.Printf("Name: %s, Host: %s, Port: %d, Socket: %s, Server Version: %s", srv.Name, srv.Host, srv.Port, srv.Socket, srv.Dist.ServerVersion)
			fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t\n", srv.Name, srv.Host, srv.Port, srv.Dist.ServerVersion)
		}
		tw.Flush()
		return nil
	},
}

func init() {
	context.RegisterCommand([]string{"add", "server"}, &addServerCmd)
	context.RegisterCommand([]string{"remove", "server"}, &removeServerCmd)
	context.RegisterCommand([]string{"list", "servers"}, &listServersCmd)
}
