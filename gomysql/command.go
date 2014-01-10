package main

import (
	"fmt"
	"io"
	"mysqld/stable"
	"strings"
	"text/wrapper"
)

// Item is a common interface for any items registered in a command
// tree.
type Item interface {
	Locate([]string) (*Command, Item, []string)
	Register([]string, Item) error
	PrintHelp(io.Writer)
	Brief() string
}

// Command is the implemention of a commands in the utility. They
// accept a context and a slice containing the arguments to the
// command. The brief description should be without a terminating
// period since it will be used in many contexts. The synopsis should
// give the arguments after the actual command: the command will be
// automatically added whenever necessary.
type Command struct {
	path, synopsis     string
	brief, description string
	body               func(*Context, []string) error
	skipStable         bool
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
	if !cmd.skipStable {
		stable, err := stable.OpenStable(ctx.Root)
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
	err := cmd.body(ctx, args)
	if err != nil {
		return err
	}

	// Write back the configuration in case the command made
	// changes to the configuration. There is no point in writing
	// back the configuration if there is no stable.
	if !cmd.skipStable {
		err := ctx.Stable.WriteConfig()
		if err != nil {
			return err
		}
	}
	return nil
}

func (cmd *Command) Locate(args []string) (*Command, Item, []string) {
	return cmd, cmd, args
}

func (cmd *Command) Register(words []string, item Item) error {
	return fmt.Errorf("Command already registered")
}

func (cmd *Command) PrintHelp(w io.Writer) {
	wrap := wrapper.New()
	fmt.Fprintf(w, "%s - %s", cmd.path, cmd.brief)
	fmt.Fprintf(w, "Usage: %s\n", cmd.synopsis)
	descr := strings.Join(wrap.Wrap(cmd.description), "\n")
	fmt.Fprintf(w, "Description:\n%s\n", descr)
}

func (cmd *Command) Brief() string {
	return cmd.brief
}

// Group is a stucture to hold information about a group of commands
// in the command structure. Each command group can contain a list of
// subgroups that can refine to commands or further to subgroups.
type Group struct {
	brief       string
	description string
	subgroup    map[string]Item
}

// Locate will return a pointer to the command matching a prefix of
// the provided arguments. If more arguments are provided than needed,
// a slice of the remaining ones are returned. If the command cannot
// be located, the best matching node is returned.
func (grp *Group) Locate(args []string) (*Command, Item, []string) {
	// If there are no arguments left and we have reached a group,
	// we cannot locate a command.
	if len(args) == 0 {
		return nil, grp, args
	}

	// Collect the candidates for matching
	candidates := []Item{}
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
func (grp *Group) Register(words []string, item Item) error {
	if len(words) == 0 {
		return fmt.Errorf("Path need to be length 1 or more")
	}

	if reg, ok := grp.subgroup[words[0]]; ok {
		if len(words) == 1 {
			return fmt.Errorf("Path already used, at %q", words[0])
		} else {
			return reg.Register(words[1:], item)
		}
	} else {
		if len(words) == 1 {
			grp.subgroup[words[0]] = item
		} else {
			return fmt.Errorf("Path is missing a group, at %q", words[0])
		}
	}
	return nil
}

func (grp *Group) PrintHelp(w io.Writer) {
	wrap := wrapper.New()
	descr := strings.Join(wrap.Wrap(grp.description), "\n")
	fmt.Fprintf(w, "Description:\n%s\n", descr)
	fmt.Fprintf(w, "Available subgroups:\n")
	for k, v := range grp.subgroup {
		fmt.Fprintf(w, "    %-14s %s\n", k, v.Brief())
	}
}

func (grp *Group) Brief() string {
	return grp.brief
}
