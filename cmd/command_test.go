package cmd

import (
	"testing"
)

func compareSlices(t *testing.T, result, expected []string) {
	if len(result) != len(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	} else {
		for i := 0; i < len(result); i++ {
			if result[i] != expected[i] {
				t.Errorf("Expected %q, but got %q", expected[i], result[i])
			}
		}
	}
}

func checkCommand(t *testing.T, tree Node, loc, remaining []string, cmd *Command) {
	c, _, as := tree.Locate(loc)
	if c != cmd {
		t.Errorf("Expected %v, got %v", cmd, c)
	} else {
		compareSlices(t, as, remaining)
	}
}

// checkGroup will test that searching for a command that is either
// incomplete or cannot be resolved because it ends up in a a group
// works as expected. The root of the command tree is provided as well
// as the command words. The expected remaining arguments are matched
// against what Locate return, and the validation function is called
// on the group to validate that it is correct.
func checkGroup(t *testing.T, tree Node, words, remaining []string, validate func(Node) bool) {
	c, n, as := tree.Locate(words)
	if c != nil {
		t.Errorf("Expected nothing, got %v", c)
	}
	if n == nil || !validate(n) {
		t.Errorf("Node %v does not validate", n)
	}
	compareSlices(t, as, remaining)
}

func TestGroup(t *testing.T) {
	tree := &Group{
		Brief:    "Just a test",
		subgroup: make(map[string]Node),
	}

	// Register a group to get some nesting tests.
	group := &Group{
		Brief:    "Another test",
		subgroup: make(map[string]Node),
	}

	cmd1 := &Command{
		Brief: "A nested command",
		Body:  func(*Context, *Command, []string) error { return nil },
	}

	if err := tree.Register([]string{"list"}, group); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if err := tree.Register([]string{"list", "second"}, cmd1); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// This should fail because the command already exists.
	if err := tree.Register([]string{"list", "second"}, cmd1); err == nil {
		t.Errorf("Expected an error, got none")
	}

	// This should fail because the path is empty
	if err := tree.Register([]string{}, cmd1); err == nil {
		t.Errorf("Expected an error, got none")
	}

	// This should fail because the path does not have groups all the way
	if err := tree.Register([]string{"list", "first", "second"}, cmd1); err == nil {
		t.Errorf("Expected an error, got none")
	}

	// Check that we can find the command
	args1 := []string{"list", "second"}
	checkCommand(t, tree, args1, args1[2:], cmd1)

	// Check that the argument count works also for simple nesting
	args2 := []string{"list", "second", "one", "two"}
	checkCommand(t, tree, args2, args2[2:], cmd1)

	// Check that we get the correct result when requesting an
	// incomplete command.
	args3 := []string{"list"}
	checkGroup(t, tree, args3, args3[1:], func(node Node) bool {
		return node.Summary() == "Another test"
	})

	// Check that the locate works correctly when requesting a
	// command that do not exist but where an initial prefix is
	// correct.
	args5 := []string{"list", "foo"}
	checkGroup(t, tree, args5, args5[1:], func(node Node) bool {
		return node.Summary() == "Another test"
	})
}

func TestCompletion(t *testing.T) {
	tree := &Group{
		Brief:    "Just a test",
		subgroup: make(map[string]Node),
	}

	// Register a group to get some nesting tests.
	listing := &Group{
		Brief:    "Listing things",
		subgroup: make(map[string]Node),
	}

	logging := &Group{
		Brief:    "Logging things",
		subgroup: make(map[string]Node),
	}

	cmd1 := &Command{
		Brief: "A nested command",
		Body:  func(*Context, *Command, []string) error { return nil },
	}

	cmd2 := &Command{
		Brief: "Something else",
		Body:  func(*Context, *Command, []string) error { return nil },
	}

	if err := tree.Register([]string{"list"}, listing); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if err := tree.Register([]string{"log"}, logging); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if err := tree.Register([]string{"list", "second"}, cmd1); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if err := tree.Register([]string{"list", "security"}, cmd2); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check that completion works
	args1 := []string{"li", "seco"}
	checkCommand(t, tree, args1, args1[2:], cmd1)

	args2 := []string{"l", "sec"}
	checkCommand(t, tree, args2, args2, nil)

	args3 := []string{"li", "sec"}
	checkCommand(t, tree, args3, args3[1:], nil)
}

func TestBasic(t *testing.T) {
	tree := &Group{
		Brief:    "Just a test",
		subgroup: make(map[string]Node),
	}

	cmd1 := &Command{
		Brief: "A command",
		Body:  func(*Context, *Command, []string) error { return nil },
	}

	// Register a command at top level and see if it can be found
	if err := tree.Register([]string{"test"}, cmd1); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	args1 := []string{"test"}
	checkCommand(t, tree, args1, args1[1:], cmd1)

	// Register an already existing command and ensure that an
	// error is returned.
	if err := tree.Register([]string{"test"}, cmd1); err == nil {
		t.Errorf("Expected an error, but didn't get one")
	}

	// Check that the number of arguments remaining after locating
	// the command works.
	args2 := []string{"test", "one", "two"}
	checkCommand(t, tree, args2, args2[1:], cmd1)

	// Check that an non-existing command is not found
	args3 := []string{"list"}
	c, _, as := tree.Locate(args3)
	if c != nil {
		t.Errorf("Expected nil, got %v", c)
	}
	if len(as) != len(args3) {
		t.Errorf("Expected %v, got %v", args3, as)
	}
}
