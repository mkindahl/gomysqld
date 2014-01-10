package main

import (
	"fmt"
	"mysqld/cmd"
	"os"
	"path/filepath"
)

var context *cmd.Context = cmd.NewContext("Available subgroups")

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
