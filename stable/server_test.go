// Copyright (c) 2013, Mats Kindahl. All rights reserved.
//
// Use of this source code is governed by a BSD license that can be
// found in the README file.

package stable

import "testing"

func TestDSN(t *testing.T) {
	var expected string
	tcp := &Server{
		User: "mats", password: "xyzzy",
		Host: "localhost", Port: 3306,
		database: "test",
	}
	expected = "mats:xyzzy@tcp(localhost:3306)/test"
	if tcp.TcpDsn() != expected {
		t.Errorf("DSN was %s, expected %s", tcp.TcpDsn(), expected)
	}

	unix := &Server{
		User: "mats", password: "xyzzy",
		Socket:   "/var/run/mysqld/mysqld.sock",
		database: "test",
	}
	expected = "mats:xyzzy@unix(/var/run/mysqld/mysqld.sock)/test"
	if unix.SocketDsn() != expected {
		t.Errorf("DSN was %s, expected %s", unix.SocketDsn(), expected)
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

	stable.Destroy()
}
