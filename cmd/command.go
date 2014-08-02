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

package cmd

import (
	"flag"
	"fmt"
	"io"
	"mysqld/stable"
	"os"
	"strings"
	"text/tabwriter"
	"text/wrapper"
)

// Node is a common interface for any items registered in a command
// tree.
type Node interface {
	Locate([]string) (*Command, Node, []string)
	Register([]string, Node) error
	PrintHelp(io.Writer)
	Summary() string
}

// Command is the implemention of a commands in the utility. They
// accept a context and a slice containing the arguments to the
// command. The brief description should be without a terminating
// period since it will be used in many contexts. The synopsis should
// give the arguments after the actual command: the command will be
// automatically added whenever necessary.
type Command struct {
	Synopsis           string
	Brief, Description string
	Body               func(*Context, *Command, []string) error
	Init               func(*Command)
	SkipStable         bool
	Flags              *flag.FlagSet

	path []string
}

// Run will run a command using a specific context. Arguments for the
// command is provided and are any words remaining after the command
// words have been removed.
//
// When executing the command, the stable in the stable root directory
// will be opened automatically, unless the skipStable flag is set.
func (cmd *Command) Run(ctx *Context, args []string) error {
	// Try to open the stable. It is OK if it cannot be opened
	// since some commands do not need it to be open.
	if !cmd.SkipStable {
		stable, err := stable.OpenStable(ctx.RootDir)
		if err != nil {
			return err
		}

		ctx.Stable = stable
		err = ctx.Stable.ReadConfig()
		if err != nil {
			return err
		}
	}

	// This execute the main body of the command with the context
	// set up properly. In case of an error, we do not write back
	// the configuration and instead just return.
	if err := cmd.Flags.Parse(args); err != nil {
		return err
	}

	err := cmd.Body(ctx, cmd, cmd.Flags.Args())
	if err != nil {
		return err
	}

	// Write back the configuration in case the command made
	// changes to the configuration. There is no point in writing
	// back the configuration if there is no stable.
	if !cmd.SkipStable {
		err := ctx.Stable.WriteConfig()
		if err != nil {
			return err
		}
	}
	return nil
}

func (cmd *Command) setup(path []string) {
	// Set up the path to the command
	cmd.path = make([]string, len(path))
	copy(cmd.path, path)

	// Create a new flag set for the command options
	cmd.Flags = flag.NewFlagSet("Options", 0)

	// Call the init function, if it was defined.
	if cmd.Init != nil {
		cmd.Init(cmd)
	}
}

func (cmd *Command) Locate(args []string) (*Command, Node, []string) {
	return cmd, cmd, args
}

func (cmd *Command) Register(words []string, node Node) error {
	return fmt.Errorf("Command already registered")
}

func (cmd *Command) Summary() string {
	return cmd.Brief
}

func (cmd *Command) PrintHelp(w io.Writer) {
	wrap := wrapper.New()
	wrap.FirstIndent = "  "
	wrap.DefaultIndent = wrap.FirstIndent

	// Command name with brief and synopsis
	pathStr := strings.Join(cmd.path, " ")
	fmt.Fprintf(w, "\n%s - %s\n\n", pathStr, cmd.Brief)
	fmt.Fprintf(w, "Usage: %s %s\n\n", pathStr, cmd.Synopsis)

	// Description
	descr := strings.Join(wrap.Wrap(cmd.Description), "\n")
	fmt.Fprintf(w, "Description:\n%s\n", descr)

	// Options
	hdrPrint := false
	tw := tabwriter.NewWriter(os.Stdout, 8, 0, 2, ' ', tabwriter.AlignRight)
	cmd.Flags.VisitAll(func(flag *flag.Flag) {
		if !hdrPrint {
			fmt.Fprintf(w, "\nOptions:\n")
			hdrPrint = true
		}

		def := ""
		if len(flag.DefValue) > 0 {
			def = fmt.Sprintf("(default %q)", flag.DefValue)
		}
		fmt.Fprintf(tw, "-%s\t%s%s\t\n", flag.Name, flag.Usage, def)
	})
	tw.Flush()
}

// Group is a stucture to hold information about a group of commands
// in the command structure. Each command group can contain a list of
// subgroups that can refine to commands or further to subgroups.
type Group struct {
	Brief       string
	Description string

	subgroup map[string]Node
	path     []string
}

// Locate will return a pointer to the command matching a prefix of
// the provided arguments. If more arguments are provided than needed,
// a slice of the remaining ones are returned. If the command cannot
// be located, the best matching node is returned.
func (grp *Group) Locate(args []string) (*Command, Node, []string) {
	// If there are no arguments left and we have reached a group,
	// we cannot locate a command.
	if len(args) == 0 {
		return nil, grp, args
	}

	// Collect the candidates for matching
	candidates := []Node{}
	for key, reg := range grp.subgroup {
		if strings.HasPrefix(key, args[0]) {
			candidates = append(candidates, reg)
		}
	}

	// If there is more than one candidate, the choice is
	// ambiguous and we cannot pick one.
	if len(candidates) != 1 {
		return nil, grp, args
	}

	c, i, as := candidates[0].Locate(args[1:])
	return c, i, as
}

// Register will allow a command or group to be registered. If it
// cannot be registered under the provided words, an error will be
// returned.
func (grp *Group) Register(words []string, node Node) error {
	if len(words) == 0 {
		return fmt.Errorf("Path need to be length 1 or more")
	}

	if reg, ok := grp.subgroup[words[0]]; ok {
		if len(words) == 1 {
			return fmt.Errorf("Path already used, at %q", words[0])
		} else {
			return reg.Register(words[1:], node)
		}
	} else {
		if len(words) == 1 {
			grp.subgroup[words[0]] = node
		} else {
			return fmt.Errorf("Path is missing a group, at %q", words[0])
		}
	}
	return nil
}

func (grp *Group) PrintHelp(w io.Writer) {
	wrap := wrapper.New()
	wrap.FirstIndent = "  "
	wrap.DefaultIndent = wrap.FirstIndent

	// Brief, with an optional path to the group name
	if len(grp.path) > 0 {
		fmt.Fprintf(w, "\n%s - %s\n\n", strings.Join(grp.path, " "), grp.Brief)
	} else {
		fmt.Fprintf(w, "\n%s\n\n", grp.Brief)
	}

	// Description
	descr := strings.Join(wrap.Wrap(grp.Description), "\n")
	fmt.Fprintf(w, "Description:\n%s\n\n", descr)

	// Print available subgroups
	fmt.Fprintf(w, "Available subgroups:\n")
	for k, v := range grp.subgroup {
		fmt.Fprintf(w, "    %-14s %s\n", k, v.Summary())
	}
}

func (grp *Group) Summary() string {
	return grp.Brief
}
