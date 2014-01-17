package main

import (
	"errors"
	"fmt"
	"mysqld/cmd"
	"mysqld/stable"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
)

var (
	ErrNoServerName   = errors.New("No server name provided")
	ErrTooManyArgs    = errors.New("Too many arguments provided")
	ErrNoFormatString = errors.New("No format string provided")
)

var srvGrp = cmd.Group{
	Brief: "Group of commands for manipulating server instances",

	Description: `All commands for manipulating and working with
	server instances are in this group.`,
}

var fmtServerCmd = cmd.Command{
	Brief: "Generate a formatted string based on server information",

	Description: `This command can be used to generate one
	formatted string for each server that matches the pattern. It
	can be used to generate information for scripts or for other
	purposes.

        The FMT is a string where any occurance of the pattern
        '{name}' will be substituted with that named field in the
        Server structure. For example, the string '{Host}:{Port}' will
        generate a host-port pair for each server.

        Each server produces a single line, so keep that in mind when
        you write your scripts.`,

	Synopsis: "FMT [PATTERN ...]",
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		if len(args) == 0 {
			return ErrNoFormatString
		}

		// Find matching servers
		servers, err := ctx.Stable.FindMatchingServers(args[1:])
		if err != nil {
			return err
		} else if len(servers) == 0 {
			return fmt.Errorf("No servers matching %q", args[1:])
		}

		// Generate the strings
		for _, srv := range servers {
			fmt.Println(srv.FormatString(args[0]))
		}

		return nil
	},
}

var addServerCmd = cmd.Command{
	Brief:    "Add a server to the stable",
	Synopsis: "NAME",

	Description: `This command will create one or more new server using a
	previously added distribution and add it to the stable.

        If a value to -dist is given, the distributions having the provided
	string as substring will be used. If less than or more than one
	distribution matching, an error will be returned. The default for the
	distribution is the empty string, which will pick every available
	distribution, which is convenient if you have only one distribution.

        If a value to -count is given, that number of servers are created from
        the distribution. The name given for the server is then a prefix rather
        than an absolute name.`,

	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		distFlag := cmd.Flags.Lookup("dist")
		countFlag := cmd.Flags.Lookup("count")

		if len(args) == 0 {
			return ErrNoServerName
		} else if len(args) > 1 {
			return ErrTooManyArgs
		}

		count, err := strconv.Atoi(countFlag.Value.String())
		if err != nil {
			return err
		}

		// Figure out the candidates for distributions
		candidates := []*stable.Dist{}
		for key, dist := range ctx.Stable.Distro {
			if strings.Contains(key, distFlag.Value.String()) {
				candidates = append(candidates, dist)
			}
		}

		if len(candidates) == 0 {
			return fmt.Errorf("No distribution containing %q", distFlag.Value.String())
		} else if len(candidates) > 1 {
			return fmt.Errorf("Ambigous choice.")
		}

		dist := candidates[0]

		// Build a list of server names to construct
		servers := []string{}
		if count == 0 {
			servers = append(servers, args[0])
		} else if count > 0 {
			for i := 1; i <= count; i++ {
				servers = append(servers, fmt.Sprintf("%s%d", args[0], i))
			}
		}

		// Create the servers
		for _, name := range servers {
			// TODO How to handle multiple errors from servers.
			if _, err := ctx.Stable.AddServer(name, dist); err != nil {
				return fmt.Errorf("Unable to create server %s: %s", name, err.Error())
			}
		}
		return nil
	},

	Init: func(cmd *cmd.Command) {
		cmd.Flags.String("dist", "", "Distribution to create the server from")
		cmd.Flags.Uint("count", 0, "Number of instances to create")
	},
}

var removeServerCmd = cmd.Command{
	Brief: "Remove a server from the stable",

	Description: `All servers matching the provided pattern will
	be removed from the stable and all associated files
	removed. Before the servers are removed, they will be
	stopped.`,

	Synopsis: "PATTERN ...",
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		if len(args) > 1 {
			return ErrTooManyArgs
		} else if len(args) == 0 {
			return ErrNoServerName
		}

		// Find matching servers
		servers, err := ctx.Stable.FindMatchingServers(args[:])
		if err != nil {
			return err
		} else if len(servers) == 0 {
			return fmt.Errorf("No servers matching %q", args[0])
		}

		// TODO How to handle multiple errors from servers.
		for _, srv := range servers {
			ctx.Stable.DelServer(srv)
		}
		return nil
	},
}

var showServersCmd = cmd.Command{
	Brief: "Show servers in the stable",

	Description: `A list of the available server instances in the
	stable is shown together with the status. The version shown is
	retrieved from the server version string shown when using
	'mysqld --version' and is extracted when the server is
	created.`,

	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		if len(args) > 0 {
			return ErrTooManyArgs
		}

		tw := tabwriter.NewWriter(os.Stdout, 8, 0, 2, ' ', tabwriter.AlignRight)
		fmt.Fprintf(tw,
			"%s\t%s\t%s\t%s\t%s\t\n",
			"NAME", "HOST", "PORT", "VERSION", "STATUS")
		for _, srv := range ctx.Stable.Server {
			fmt.Fprintf(tw,
				"%s\t%s\t%d\t%s\t%s\t\n",
				srv.Name, srv.Host, srv.Port,
				srv.Dist.ServerVersion, srv.Status())
		}
		tw.Flush()
		return nil
	},
}

var startServerCmd = cmd.Command{
	Brief: "Start a server",

	Description: `All servers matching the provided will be started in the
	background. If any options are provided in addition to the name, they
	will be added to the list of options when starting the server.`,

	Synopsis: "PATTERN OPTION ...",
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		if len(args) == 0 {
			return ErrNoServerName
		}

		// Fetch the server from the stable
		servers, err := ctx.Stable.FindMatchingServers(args[:1])
		if err != nil {
			return err
		} else if len(servers) == 0 {
			return fmt.Errorf("No servers matching %q", args[0])
		}

		// TODO How to handle multiple errors from servers.
		for _, srv := range servers {
			// Check if the server is running, i.e., if there is a PID file
			if srv.Status() == stable.SERVER_RUNNING {
				return fmt.Errorf("Server %q already running", srv.Name)
			}

			// Time to do the daemonize fandango
			argv := []string{
				filepath.Base(srv.BinPath),
				fmt.Sprintf("--defaults-file=%s", srv.ConfigFile),
			}
			argv = append(argv, args[1:]...)
			forkDaemon(srv.BinPath, srv.BaseDir, srv.LogPath, argv)
		}
		return nil
	},
}

var stopServerCmd = cmd.Command{
	Brief: "Stop a server",

	Description: `All servers matching the pattern will be stopped by
	sending TERM (11) to it. This is the normal shutdown procedure for a
	graceful shutdown of a server, but it only work when done on the local
	machine. If an attempt to shut down a server on a remote machine is
	done, an error will currently be thrown.`,

	Synopsis: "PATTERN",
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		if len(args) == 0 {
			return ErrNoServerName
		} else if len(args) > 1 {
			return ErrTooManyArgs
		}

		// Fetch matching servers from the stable
		servers, err := ctx.Stable.FindMatchingServers(args[:1])
		if err != nil {
			return err
		} else if len(servers) == 0 {
			return fmt.Errorf("No servers matching %q", args[0])
		}

		// TODO How to handle multiple errors from servers.
		for _, srv := range servers {
			if !srv.IsLocal() {
				return fmt.Errorf("Non-local server: server is at %s", srv.Host)
			}

			// TODO: Check that the server is local
			if srv.Status() != stable.SERVER_RUNNING {
				return fmt.Errorf("Server %s not running", srv.Name)
			}

			if pid, err := srv.Pid(); err != nil {
				return fmt.Errorf("Server %s: %s", srv.Name, err)
			} else {
				syscall.Kill(pid, syscall.SIGTERM)
			}
		}
		return nil
	},
}

// forkDaemon will start a server as a daemon. The path to the binary
// is given in binPath, the directory where the server should run is
// given in runDir, and the path where the standard output and
// standard error will be directed is given by outPath. Note that the
// outPath will be opened in append mode, and created if it does not
// exists.
func forkDaemon(binPath, runDir, outPath string, argv []string) error {
	pid, _, errno := syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)
	if errno != 0 {
		return fmt.Errorf("Failed to fork: %s", errno.Error())
	}

	// Parent process just return.
	if pid > 0 {
		// TODO Do we need to check that the start succeeded? Create a
		// pipe to communicate over then.
		return nil
	}

	// In child process
	var file *os.File
	var err error

	os.Chdir(runDir)

	// Re-direct standard error and standard output to logfile
	file, err = os.OpenFile(outPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err == nil {
		fd := file.Fd()
		syscall.Dup2(int(fd), int(os.Stdout.Fd()))
		syscall.Dup2(int(fd), int(os.Stderr.Fd()))
	} else {
		return err
	}

	// Re-direct standard input to /dev/null
	file, err = os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err == nil {
		syscall.Dup2(int(file.Fd()), int(os.Stdin.Fd()))
	} else {
		return err
	}

	if err := syscall.Exec(binPath, argv, os.Environ()); err != nil {
		return err
	}

	return nil
}

func init() {
	context.RegisterGroup([]string{"server"}, &srvGrp)
	context.RegisterCommand([]string{"server", "add"}, &addServerCmd)
	context.RegisterCommand([]string{"server", "remove"}, &removeServerCmd)
	context.RegisterCommand([]string{"server", "show"}, &showServersCmd)
	context.RegisterCommand([]string{"server", "start"}, &startServerCmd)
	context.RegisterCommand([]string{"server", "stop"}, &stopServerCmd)
	context.RegisterCommand([]string{"server", "fmt"}, &fmtServerCmd)
}
