package main

import (
	"fmt"
	"io"
	"log"
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
	Where Item
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
	Root   string
	tree   Item
	Stable *stable.Stable
}

// NewContext will create a new context.
func NewContext() *Context {
	context := &Context{
		tree: &Group{
			brief:    "Basic commands",
			subgroup: make(map[string]Item),
		},
	}

	return context
}

// RegisterCommand will register a new command under the given
// sequence of words. Each word before the last one is expected to
// hold a group, while the last word should not be registered for the
// group.
func (ctx *Context) RegisterCommand(words []string, cmd *Command) error {
	return ctx.tree.Register(words, cmd)
}

// RegisterGroup will register a new group under the given sequence of
// words. Each word before the last one is expected to hold a group,
// while the last word should not be registered for the group.
func (ctx *Context) RegisterGroup(words []string, group *Group) error {
	if group.subgroup == nil {
		group.subgroup = make(map[string]Item)
	}
	return ctx.tree.Register(words, group)
}

// RunCommand will run the command given by the words. In the event of
// a failure, a run error is returned containing information about the
// error and where the failure occured.
func (ctx *Context) RunCommand(words []string) *RunError {
	log.Printf("words: %v, len(words): %d", words, len(words))

	// Locate the command and compute the arguments to the command
	// by recursively going through the command tree.
	cmd, node, args := ctx.tree.Locate(words)
	if cmd == nil {
		// Find the first unrecognized word, or if all words
		// are recognized, find the last word in the list.
		end := len(words) - len(args)
		if end < cap(words) {
			end++
		}
		err := fmt.Errorf("Command not found: %q\n", strings.Join(words[:end], " "))
		return &RunError{Err: err, Where: node}
	}

	if err := cmd.Run(context, args); err != nil {
		return &RunError{Err: err, Where: node}
	}
	return nil
}
