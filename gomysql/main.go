// Copyright (c) 2014, Oracle and/or its affiliates. All rights reserved.

// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; version 2 of the License.

// This program is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program; if not, write to the Free Software
// Foundation, Inc., 51 Franklin St, Fifth Floor, Boston, MA 02110-1301
// USA

// Utility to manage several MySQL servers for experiments.
//
// This utility manages a *stable* of MySQL servers that can be
// created, started, stopped, and manipulated in other ways to perform
// experiments on them.
//
// You can always get help by using the help command, for example:
//
//    gomysql help init
//    gomysql help server
//    gomysql help server add
//
// To create a MySQL stable in the current directory, use the command:
//
//    gomysql init .
//
// To be able to experiment with multiple versions of servers, the
// stable can maintain servers built from different binary
// distributions. Each stable therefore contain one or more
// *distributions* from which one or more *servers* can be created.
//
// To add a new distribution to the stable, you use the command
// 'distribution add'. All commands can be abbreviated, as long as
// they can be uniquely identified, so:
//
//    gomysql dist add mysql-5.6.14-linux-glibc2.5-i686.tar.gz
//
// The tar file is a binary distribution and will contain all the
// files necessary to run the server. This command will unpack the
// binary distribution into the stable and add it as a distribution
// under the name of the directory that the tar file unpacks to.
//
// Once you have added a distribution to the stable, you can create
// new servers from it. When creating servers, a distribution is
// needed. If you have a single distribution in your stable, it will
// automatically be used, so you can create a new named server using:
//
//    gomysql server add my_server
//
// If you have several distributions added to the stable, you can give
// the distribution that should be used using the -dist option. The
// value should be a substring of one of the distribution names. This
// make is easy to pick the right distribution by just providing, for
// example, the version:
//
//    gomysql server add -dist=5.6.14 my_server
//
// The command will create a server data directory and bootstrap the
// server so that you can use it. Occationally, you want to create a
// bunch of servers instead of a single one. Since it is quite tedius
// to write the 'server add' command if you want to and multiple
// servers, there is a -count option to the 'server add' command. When
// the -count option is used, the name provided last ('slave.' in this
// case) is used as a prefix to the server names, so this command will
// create servers 'slave.1', ..., 'slave.10':
//
//    gomysql server add -dist=5.6.14 -count=10 slave.
//
// Starting and stopping the servers is as easy as:
//
//    gomysql server start slave.*
//    gomysql server stop slave.*
//
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
		log.SetPriority(log.Priority(flagLevel))

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
	flag.IntVar(&flagLevel, "level", log.PRIORITY_WARNING, "Logging level (0: error, 1: warnings, 2: info, 3: debug)")
}
