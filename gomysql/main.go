package main

import (
	"flag"
	"fmt"
	"mysqld/cmd"
	"mysqld/log"
	"os"
	"path/filepath"
)

var flagRoot string
var flagLevel int

var brief = "Utility for managing a stable of MySQL servers"

var description = `Easy creation and distribution of MySQL
servers. The utility support running different versions of servers at
the same time.

To create a new stable, use 'init'.

To add a distribution, use 'distribution add'.

To get help on a command, use the 'help <command>'.`

var context *cmd.Context = cmd.NewContext(brief, description)

func main() {
	flag.Parse()

	if args := flag.Args(); len(args) == 0 {
		flag.Usage()
		os.Exit(2)
	} else {
		prog := filepath.Base(os.Args[0])
		context.RootDir = flagRoot
		log.SetLevel(log.LogLevel(flagLevel))

		if err := context.RunCommand(args); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", prog, err)
			os.Exit(2)
		}
	}
}

func usage() {
	prog := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "Usage: %s [ <options> ] <word> ...\n", prog)
	fmt.Fprintf(os.Stderr, "\nGlobal options:\n")
	flag.PrintDefaults()
	context.PrintHelp(os.Stderr)
}

func init() {
	flag.Usage = usage
	flag.StringVar(&flagRoot, "root", ".", "Root directory for stable")
	flag.IntVar(&flagLevel, "level", log.LOGLEVEL_WARNING, "Logging level (0: error, 1: warnings, 2: info, 3: debug)")
}
