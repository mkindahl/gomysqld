package main

import (
	"flag"
	"fmt"
	"mysqld/cmd"
	"os"
	"path/filepath"
)

var flagRoot string

var brief = "Utility for managing a stable of MySQL servers"

var description = `Easy creation and distribution of MySQL
servers. The utility support running different versions of servers at
the same time.

To get help, use the 'help' command.`

var context *cmd.Context = cmd.NewContext(brief, description)

func main() {
	context.RootDir = flagRoot
	prog := filepath.Base(os.Args[0])

	if len(os.Args) == 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> ...\n", prog)
		context.Top.PrintHelp(os.Stderr)
		os.Exit(2)
	}

	if err := context.RunCommand(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", prog, err)
		os.Exit(2)
	}
}

func init() {
	flag.StringVar(&flagRoot, "root", ".", "Root directory for stable")
}
