package main

import (
	"fmt"
	"os"
	"path/filepath"
)

var context *Context = NewContext()

func main() {
	context.Root = flagRoot
	prog := filepath.Base(os.Args[0])

	if len(os.Args) == 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> ...\n", prog)
		context.tree.PrintHelp(os.Stderr)
		os.Exit(2)
	}

	if err := context.RunCommand(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", prog, err)
		os.Exit(2)
	}
}
