// Copyright (c) 2013, Mats Kindahl. All rights reserved.
//
// Use of this source code is governed by a BSD license that can be
// found in the README file.

package stable

import (
	"testing"
)

func TestDSN(t *testing.T) {
	var expected string
	tcp := &Server{
		User:     "mats",
		Password: "xyzzy",
		Host:     "localhost",
		Port:     3306,
		database: "test",
	}
	expected = "mats:xyzzy@tcp(localhost:3306)/test"
	if tcp.TcpDsn() != expected {
		t.Errorf("DSN was %s, expected %s", tcp.TcpDsn(), expected)
	}

	unix := &Server{
		User:     "mats",
		Password: "xyzzy",
		Socket:   "/var/run/mysqld/mysqld.sock",
		database: "test",
	}
	expected = "mats:xyzzy@unix(/var/run/mysqld/mysqld.sock)/test"
	if unix.SocketDsn() != expected {
		t.Errorf("DSN was %s, expected %s", unix.SocketDsn(), expected)
	}
}

func TestFormatString(t *testing.T) {
	srv := &Server{
		User:     "mats",
		Password: "xyzzy",
		Host:     "localhost",
		Port:     3306,
		database: "test",
	}
	expect := "This is just localhost on port 3306"
	result := srv.FormatString("This is just {Host} on port {Port}")
	if result != expect {
		t.Errorf("Expected %q, got %q", expect, result)
	}
}

func TestAddServer(t *testing.T) {
	if len(flagDist) == 0 {
		t.Skip("No distribution provided with -dist flag, skipping test")
	}

	stable, err := CreateStable(".")
	if err != nil {
		t.Fatalf("Failed to create stable: %q", err)
	}

	dist, err := stable.AddDist(flagDist)
	if err != nil {
		t.Fatalf("Failed to add distribution: %q", err)
	}

	server, err := stable.AddServer("my_server", dist)
	if err != nil {
		t.Errorf("Failed to create server: %q", err)
	} else if server == nil {
		t.Errorf("No server returned and no error")
	} else if server.Name != "my_server" {
		t.Errorf("Server name was %q, expected %q", server.Name, "my_server")
	}

	srv, ok := stable.Server["my_server"]
	if !ok || srv != server {
		t.Errorf("Server %q not added correctly to server list", server.Name)
	}
	if srv.PidPath != srv.run("mysqld.pid") {
		t.Errorf("PidPath should be %q, was %q", srv.PidPath, srv.run("mysqld.pid"))
	}

	stable.Destroy()
}
