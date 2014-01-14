// Package that support command trees and allow you to have a
// hierarchy of commands and register groups of commands in a way
// similar to how GDB works.
//
// The commands are separated into groups, where each group can
// contain either subgroups or specicif commands. This allow you to
// add command hierarchies such as "show servers" (where "show" is a
// group and "show servers" is the real command).
package cmd

import (
	"fmt"
	"io"
	"mysqld/stable"
	"strings"
)

// RunError record an error and information about where the error were
// generated. It can be used to print an apropriate error message
// depending on where the error was detected. It satisfies the
// requirements of the error interface by providing the Error
// function.
type RunError struct {
	Err   error
	Where Node
}

func (err *RunError) Error() string {
	return err.Err.Error()
}

// PrintHelp will write the error followed by a context-dependent help
// message on the writer w.
func (err *RunError) PrintHelp(w io.Writer) {
	if err.Where != nil {
		err.Where.PrintHelp(w)
	}
}

// Context hold the structure of commands, including such things as
// the complete list of all commands (as a tree), the Stable they are
// running in, the root directory, etc. Each command above receive the
// context when executing so that they can look up such items while
// executing.
type Context struct {
	RootDir string
	Top     *Group
	Stable  *stable.Stable
}

// NewContext will create a new context.
func NewContext(summary, description string) *Context {
	context := &Context{
		Top: &Group{
			Brief:       summary,
			Description: description,
			subgroup:    make(map[string]Node),
		},
	}

	return context
}

// RegisterCommand will register a new command under the given
// sequence of words. Each word before the last one is expected to
// hold a group, while the last word should not be registered for the
// group.
func (ctx *Context) RegisterCommand(words []string, cmd *Command) {
	err := ctx.Top.Register(words, cmd)
	if err == nil {
		cmd.setup(words)
	} else {
		panic(err.Error())
	}
}

// RegisterGroup will register a new group under the given sequence of
// words. Each word before the last one is expected to hold a group,
// while the last word should not be registered for the group.
func (ctx *Context) RegisterGroup(words []string, grp *Group) {
	if grp.subgroup == nil {
		grp.subgroup = make(map[string]Node)
	}
	err := ctx.Top.Register(words, grp)
	if err == nil {
		grp.path = make([]string, len(words))
		copy(grp.path, words)
	} else {
		panic(err.Error())
	}
}

// Locate a command given a sequence of words.
//
// If a command is successfully found, a pointer to it will be
// returned together with the node (this will always be a command) and
// the remaining words.
//
// If the command is not found, nil will be returned together with the
// node containing the first mismatch (this will always be a group)
// and the remaining words that could not be matched.
func (ctx *Context) Locate(words []string) (*Command, Node, []string) {
	return ctx.Top.Locate(words)
}

// RunCommand will run the command given by the words. In the event of
// a failure, a run error is returned containing information about the
// error and where the failure occured.
func (ctx *Context) RunCommand(words []string) *RunError {
	// Locate the command and compute the arguments to the command
	// by recursively going through the command tree.
	cmd, node, args := ctx.Top.Locate(words)
	if cmd == nil {
		// Find the first unrecognized word, or if all words
		// are recognized, find the last word in the list.
		end := len(words) - len(args)
		if end < cap(words) {
			end++
		}
		err := fmt.Errorf("Command not found: %q", strings.Join(words[:end], " "))
		return &RunError{Err: err, Where: node}
	}

	if err := cmd.Run(ctx, args); err != nil {
		return &RunError{Err: err, Where: node}
	}
	return nil
}