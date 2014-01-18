// Copyright (c) 2013, Mats Kindahl. All rights reserved.
//
// Use of this source code is governed by a BSD license that can be
// found in the README file.

package stable

import (
	"flag"
	"mysqld/log"
	"os"
	"path/filepath"
	"testing"
)

func TestPathType(t *testing.T) {
	if pathType := pathType("foo/mysql-9.9.9.tar.gz"); pathType != TGZ_PATH {
		t.Errorf("Path type %v expected, was %v", pathType, TGZ_PATH)
	}
	if pathType := pathType("mysql-9.9.9.tar.gz"); pathType != TGZ_PATH {
		t.Errorf("Path type %v expected, was %v", pathType, TGZ_PATH)
	}
	if pathType := pathType("mysql-9.9.9.tar"); pathType != TAR_PATH {
		t.Errorf("Path type %v expected, was %v", pathType, TAR_PATH)
	}
	if pathType := pathType("foo/mysql-9.9.9.tar"); pathType != TAR_PATH {
		t.Errorf("Path type %v expected, was %v", pathType, TAR_PATH)
	}
	if pathType := pathType("foo/mysql-9.9.9.zip"); pathType != ZIP_PATH {
		t.Errorf("Path type %v expected, was %v", pathType, ZIP_PATH)
	}
	if pathType := pathType("mysql-9.9.9.zip"); pathType != ZIP_PATH {
		t.Errorf("Path type %v expected, was %v", pathType, ZIP_PATH)
	}
	os.Mkdir("mysql-9.9.9", 0755)
	if pathType := pathType("mysql-9.9.9"); pathType != DIR_PATH {
		t.Errorf("Path type %v expected, was %v", pathType, DIR_PATH)
	}
	os.Remove("mysql-9.9.9")
	os.MkdirAll("foo/mysql-9.9.9", 0755)
	if pathType := pathType("foo/mysql-9.9.9"); pathType != DIR_PATH {
		t.Errorf("Path type %v expected, was %v", pathType, DIR_PATH)
	}
	os.RemoveAll("foo")
	if pathType := pathType("mysql-9.9.9"); pathType != UNKNOWN_PATH {
		t.Errorf("Path type %v expected, was %v", pathType, UNKNOWN_PATH)
	}
}

func TestParseVersionString(t *testing.T) {
	strings := map[string]string{
		"5.5.32-0ubuntu0.12.04.1-log": "mysqld  Ver 5.5.32-0ubuntu0.12.04.1-log for debian-linux-gnu on i686 ((Ubuntu)))",
		"5.6.14":                      "mysql-5.6.14-linux-glibc2.5-i686/bin/mysqld  Ver 5.6.14 for linux-glibc2.5 on i686 (MySQL Community Server (GPL))",
	}

	for expected, versionString := range strings {
		dist := &Dist{}
		dist.parseVersionString(versionString)
		if dist.ServerVersion != expected {
			t.Errorf("Expected version string %q, found %q", expected, dist.ServerVersion)
		}
	}
}

func TestScanVersionFile(t *testing.T) {
	files := map[string]string{
		"include_1.h": "5.1.71",
		"include_2.h": "5.1.5",
	}
	for file, version := range files {
		fi, err := os.Open(filepath.Join("datafiles", file))
		if err != nil {
			t.Skipf("Cannot open %q, skipped", file)
			continue
		}
		dist := &Dist{}
		dist.scanVersionFile(fi)
		if dist.Version != version {
			t.Errorf("Expected version %v, found version %v", version, dist.Version)
		}
	}
}

var flagDist, flagVersion string

func init() {
	flag.StringVar(&flagDist, "dist", "", "Distribution to use for test")
	flag.StringVar(&flagVersion, "version", "", "Version expected for the distribution")
}

func TestDistSetup(t *testing.T) {
	if len(flagDist) == 0 {
		t.Skip("No distribution provided with -dist flag, skipping test")
	}

	if len(flagVersion) == 0 {
		t.Skip("No expected version provided with -version flag, skipping test")
	}

	stable, err := CreateStable(".")
	if err != nil {
		t.Errorf("CreateStable: error returned, none expected: %v", err.Error())
		return
	}

	if _, err := stable.AddDist("invalid-name"); err == nil {
		t.Errorf("AddDist: expected error, didn't got one")
		return
	}

	dist, err := stable.AddDist(flagDist)

	if err != nil {
		t.Fatalf("Error returned, none expected:\n%s", err.Error())
	}

	if dist.Version != flagVersion {
		t.Errorf("Version %v expected, was %v", flagVersion, dist.Version)
	}

	log.Debugf("Distribution with name %q created", dist.Name)
	if len(dist.Name) == 0 {
		t.Errorf("Name was expected, none assigned")
	}

	if len(stable.Distro) != 1 {
		t.Errorf("Number of registered distributions is %v, expected 1", len(stable.Distro))
	}

	if stable.Distro[dist.Name] != dist {
		t.Errorf("Distribution not registered correctly")
	}

	stable.Destroy()
}
