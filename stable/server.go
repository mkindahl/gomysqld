package stable

import (
	"fmt"
	"io"
	"log"
	"mysqld/cnf"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// Server structure contain all information about a server.
type Server struct {
	Name, Host, Socket           string
	BaseDir, DataDir, ConfigFile string
	ServerId, Port               int
	Options                      *cnf.Config
	User, password, database     string
	Dist                         *Dist
}

// bin will return the path to the name of a binary for the server.
func (srv *Server) bin(name string) string {
	return filepath.Join(srv.Dist.Root, "bin", name)
}

// log will return the path to a name in the log directory for the
// server.
func (srv *Server) log(name string) string {
	return filepath.Join(srv.BaseDir, "log", name)
}

// tmp will return the path to a name in the tmp directory for the
// server.
func (srv *Server) tmp(name string) string {
	return filepath.Join(srv.BaseDir, "tmp", name)
}

// run will return the path to a name in the run directory for the
// server.
func (srv *Server) run(name string) string {
	return filepath.Join(srv.BaseDir, "run", name)
}

// sqlFiles list the files necessary for bootstrapping a fresh server.
var sqlFiles = []string{
	"share/mysql_system_tables.sql",
	"share/mysql_system_tables_data.sql",
	"share/mysql_test_data_timezone.sql",
	"share/fill_help_tables.sql",
}

// createBootstrap will create a bootstrap file for the server.
func (srv *Server) writeBootstrapFile(bs *os.File) error {
	log.Printf("Creating bootstrap file %q...\n", bs.Name())

	// Write the header to the bootstrap file
	header := []string{
		"SET SESSION SQL_LOG_BIN = 0;",
		"CREATE DATABASE IF NOT EXISTS mysql;",
		"CREATE DATABASE IF NOT EXISTS test;",
		"USE mysql;",
	}
	for _, line := range header {
		fmt.Fprintln(bs, line)
	}

	// Append bootstrap files from distribution
	for _, fname := range sqlFiles {
		fullname := filepath.Join(srv.Dist.Root, fname)
		rd, err := os.Open(fullname)
		if err != nil {
			return err
		}
		//		log.Printf("Appending %v\n", fullname)
		_, err = io.Copy(bs, rd)
		rd.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (srv *Server) bootstrap() error {
	bsName := srv.tmp("bootstrap.sql")
	if bs, err := os.Create(bsName); err == nil {
		err = srv.writeBootstrapFile(bs)
		bs.Close()
		if err != nil {
			return err
		}
	} else {
		return err
	}

	// Open the bootstrap file
	bsSql, err := os.Open(bsName)
	if err != nil {
		return err
	}

	// Run the bootstrap command
	bsLog, err := os.Create(srv.log("bootstrap.log"))
	cnfOpt := fmt.Sprintf("--defaults-file=%s", srv.ConfigFile)
	cmd := exec.Command(srv.bin("mysqld"), cnfOpt, "--bootstrap")
	cmd.Stdin = bsSql
	cmd.Stdout = bsLog
	cmd.Stderr = bsLog
	log.Print("Bootstrapping ", cmd.Args)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// createServer will create and populate the server structure with the
// correct information.
func (stable *Stable) newServer(name string, dist *Dist) (*Server, error) {
	// Collect all the information
	baseDir := filepath.Join(stable.serverDir, name)
	dataDir := filepath.Join(baseDir, "data")
	runDir := filepath.Join(baseDir, "run")
	cnfFile := filepath.Join(baseDir, "my.cnf")
	pidFile := filepath.Join(runDir, "mysqld.pid")
	socket := filepath.Join(runDir, "mysqld.sock")
	port := stable.fetchPortNumber()
	serverId := stable.fetchServerId()

	// Create the server instances
	server := &Server{
		Name:       name,
		BaseDir:    baseDir,
		DataDir:    dataDir,
		ConfigFile: cnfFile,
		Host:       "localhost",
		Port:       port,
		Socket:     socket,
		ServerId:   serverId,
		Options:    cnf.New(),
		Dist:       dist,
	}

	// Create and fill in default options
	option := map[string]map[string]string{
		"mysqladmin": map[string]string{
			"socket": server.Socket,
			"user":   "root",
			"host":   server.Host,
			"port":   strconv.Itoa(server.Port),
		},

		"mysqld": map[string]string{
			"basedir":   baseDir,
			"datadir":   dataDir,
			"socket":    server.Socket,
			"port":      strconv.Itoa(server.Port),
			"pid_file":  pidFile,
			"server_id": strconv.Itoa(serverId),
		},

		"mysql": map[string]string{
			"protocol": "tcp",
			"host":     server.Host,
			"port":     strconv.Itoa(server.Port),
			"prompt":   "'" + name + "> '",
		},
	}

	// Set up the language configuration correctly for the version of the server.
	if dist.Version <= "5.5.0" {
		option["mysqld"]["language"] = filepath.Join(dist.Root, "share", "english")
	} else {
		option["mysqld"]["lc_messages_dir"] = filepath.Join(dist.Root, "share")
		option["mysqld"]["lc_messages"] = "en_US"
	}

	server.Options.Import(option)

	return server, nil
}

// create will create all the necessary directories and files for a
// fully functional server. In the event of an error, no files will be
// cleaned up: that is the responsibility of the caller.
func (srv *Server) setup(stable *Stable) error {
	// Create all the necessary directories
	dirs := []string{
		srv.BaseDir,
		srv.DataDir,
		filepath.Join(srv.BaseDir, "run"),
		filepath.Join(srv.BaseDir, "log"),
		filepath.Join(srv.BaseDir, "tmp"),
	}

	for _, dir := range dirs {
		//		log.Printf("Creating server directory %q\n", dir)
		if err := os.Mkdir(dir, 0755); err != nil {
			return err
		}
	}

	cnfFile := filepath.Join(srv.BaseDir, "my.cnf")
	if fd, err := os.Create(cnfFile); err != nil {
		return err
	} else {
		//		log.Printf("Writing configuration file %q\n", cnfFile)
		srv.Options.Write(fd)
		fd.Close()
	}

	return nil
}

// teardown is executed to tear down the directory structure for the
// server. If the server is running, an error is returned.
func (srv *Server) teardown() error {
	// TODO: Check that the server is not running
	return os.RemoveAll(srv.BaseDir)
}

// AddServer will add a new server to the stable under a name. If the
// server was created successfully, it will be returned. If it failed
// for some reason, nil will be returned and the error that caused the
// creation to fail.
func (stable *Stable) AddServer(name string, dist *Dist) (*Server, error) {
	// Create the in-memory server structure
	server, err := stable.newServer(name, dist)
	if err != nil {
		return nil, err
	}

	// Create the necessary files and directories
	if err := server.setup(stable); err != nil {
		return nil, err
	}

	// Bootstrap the server
	if err := server.bootstrap(); err != nil {
		os.RemoveAll(server.BaseDir)
		return nil, err
	}

	stable.Server[name] = server

	return server, nil
}

// DelServerByName will delete the server given by the name. The
// complete server will be removed by removing all server files and it
// will not be possible to recover the server after this. If no server
// exists by that name, an error will be returned.
func (stable *Stable) DelServerByName(name string) error {
	if srv, exists := stable.Server[name]; !exists {
		return fmt.Errorf("No server named %q exists", name)
	} else {
		return stable.DelServer(srv)
	}
}

// Delete
func (stable *Stable) DelServer(srv *Server) error {
	if err := srv.teardown(); err != nil {
		return err
	}
	
	delete(stable.Server, srv.Name)
	return nil
}


func (s *Server) SocketDsn() string {
	return fmt.Sprintf("%v:%v@unix(%v)/%v", s.User, s.password, s.Socket, s.database)
}

func (s *Server) TcpDsn() string {
	return fmt.Sprintf("%v:%v@tcp(%v:%v)/%v", s.User, s.password, s.Host, s.Port, s.database)
}
