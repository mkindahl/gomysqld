// Copyright (c) 2013, Mats Kindahl. All rights reserved.
//
// Use of this source code is governed by a BSD license that can be
// found in the README file.

package stable

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStableSetup(t *testing.T) {
	t.Log("Creating stable")
	stable, err := CreateStable(".")
	if err != nil {
		t.Fatal("Stable exists, aborting")
	}

	// Check that the stable root is what was expected
	t.Log("Checking that root is what is expected")
	cwd, _ := os.Getwd()
	expected := filepath.Join(cwd, STABLE_DIR)
	if stable.Root != expected {
		t.Errorf("Root is %s, expected %s", stable.Root, expected)
	}

	// Check that trying to overwrite an existing stable fails
	t.Log("Trying to create a stable when one exists should fail")
	_, err = CreateStable(".")
	if err == nil {
		t.Errorf("CreateStable: error expected, none returned\n")
	}

	// Check that the stable contain the expected directories
	dirs := map[string]string{
		"stable":       stable.Root,
		"distribution": filepath.Join(stable.Root, "dist"),
		"server":       filepath.Join(stable.Root, "server"),
	}

	for name, dir := range dirs {
		stat, err := os.Stat(dir)
		if err != nil || !stat.IsDir() {
			t.Errorf("Missing %v directory", name)
		}
	}

	if err := stable.Destroy(); err != nil {
		t.Errorf("Destroy: Got error %v", err)
	}
}

func stablesEqual(t *testing.T, stable1, stable2 *Stable) bool {
	stable1str, err := json.Marshal(stable1)
	if err != nil {
		t.Errorf("Marshal of LHS failed: %s", err)
	}

	stable2str, err := json.Marshal(stable2)
	if err != nil {
		t.Errorf("Marshal of RHS failed: %s", err)
	}

	// t.Logf("JSON LHS: `%s`\n", stable1str)
	// t.Logf("JSON RHS: `%s`\n", stable2str)

	return string(stable1str) == string(stable2str)
}

func TestConfig(t *testing.T) {
	// Create an empty stable
	stable1, err := CreateStable(".")
	if err != nil {
		t.Fatalf("Unable to create stable: %s", err)
	}

	defer stable1.Destroy()

	// Build a second reference to the stable to see that we can
	// read from it.
	stable2, err := OpenStable(".")
	if err != nil {
		t.Fatalf("Unable to open stable: %s", err)
	}

	if !stablesEqual(t, stable1, stable2) {
		t.Errorf("Stables not equal")
	}

	// Add a fake distribution to see that changes are reflected.
	dist, _ := stable1.newDist()
	dist.Name = "my_test"
	stable1.Distro[dist.Name] = dist
	stable1.WriteConfig()
	stable2.ReadConfig()

	if !stablesEqual(t, stable1, stable2) {
		t.Errorf("Stables not equal")
	}
}
