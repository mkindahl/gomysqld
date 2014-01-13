package main

import (
	"errors"
	"fmt"
	"mysqld/cmd"
	"mysqld/stable"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"
)

var (
	ErrNoServerName = errors.New("No server name provided")
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
	previously added distribution and add it to the stable. If a
	value to -dist is given, the distributions having the provided
	string as substring will be used. If less than or more than
	one distribution matching, an error will be returned. The
	default for the distribution is the empty string, which will
	pick every available distribution, which is convenient if you
	have only one distribution.`,

	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		if len(args) != 1 {
			return ErrNoServerName
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
	Brief: "Remove a server from the stable",

	Description: `The named server will be removed from the stable
	and all associated files removed.`,

	Synopsis: "NAME",
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		if len(args) != 1 {
			return ErrNoServerName
		}

		srv, ok := ctx.Stable.Server[args[0]]
		if !ok {
			return fmt.Errorf("No such server: %s", args[0])
		}

		return ctx.Stable.DelServer(srv)
	},
}

var showServersCmd = cmd.Command{
	Brief: "Show servers in the stable",

	Description: `A list of the available server instances in the
	stable is shown together with the status.`,

	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		if len(args) > 0 {
			argStr := strings.Join(args, " ")
			return fmt.Errorf("Wrong number of arguments %q", argStr)
		}
		tw := tabwriter.NewWriter(os.Stdout, 8, 0, 2, ' ', tabwriter.AlignRight)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t\n", "NAME", "HOST", "PORT", "VERSION", "STATUS")
		for _, srv := range ctx.Stable.Server {
			fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\t\n", srv.Name, srv.Host, srv.Port, srv.Dist.ServerVersion, srv.Status())
		}
		tw.Flush()
		return nil
	},
}

var startServerCmd = cmd.Command{
	Brief: "Start a server",

	Description: `The named server will be started in the
	background. If any options are provided in addition to the
	name, they will be added to the list of options when starting
	the server.`,

	Synopsis: "NAME OPTION ...",
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		if len(args) == 0 {
			return ErrNoServerName
		}

		// Fetch the server from the stable
		srv, ok := ctx.Stable.Server[args[0]]
		if !ok {
			return fmt.Errorf("No such server: %s", args[0])
		}
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
		return forkDaemon(srv.BinPath, srv.BaseDir, srv.LogPath, argv)
	},
}

var stopServerCmd = cmd.Command{
	Brief: "Stop a server",

	Description: `The server will be stopped by sending TERM (11)
	to it. This is the normal shutdown procedure for a graceful
	shutdown of a server, but it only work when done on the local
	machine. If an attempt to shut down a server on a remote
	machine is done, an error will currently be thrown.`,

	Synopsis: "NAME",
	Body: func(ctx *cmd.Context, cmd *cmd.Command, args []string) error {
		if len(args) != 1 {
			return ErrNoServerName
		}

		// Fetch the server from the stable
		srv, ok := ctx.Stable.Server[args[0]]
		if !ok {
			return fmt.Errorf("No such server: %s", args[0])
		}

		if !srv.IsLocal() {
			return fmt.Errorf("Non-local server: server is at %s", srv.Host)
		}

		// TODO: Check that the server is local
		if srv.Status() != stable.SERVER_RUNNING {
			return fmt.Errorf("Server %q not running", srv.Name)
		}

		if pid, err := srv.Pid(); err != nil {
			return err
		} else {
			return syscall.Kill(pid, syscall.SIGTERM)
		}
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
}
