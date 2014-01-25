package stable

import (
	"fmt"
	"io"
	"mysqld/cnf"
	"mysqld/log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Status is the status of a server. It overloads the String()
// function to be possible to use in contexts requiring a string.
type Status int

var statusString = []string{
	"Stopped",
	"Running",
}

func (s Status) String() string {
	return statusString[s]
}

const (
	SERVER_UNAVAIL = iota
	SERVER_RUNNING
)

// Server structure contain all information about a server.
type Server struct {
	Name, Host, Socket        string
	BaseDir, DataDir          string
	ConfigFile                string
	BinPath, LogPath, PidPath string
	ServerId, Port            int
	Options                   *cnf.Config
	User, Password, database  string
	Dist                      *Dist
}

func (srv *Server) String() string {
	return srv.Name
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
	"mysql_system_tables.sql",
	"mysql_system_tables_data.sql",
	"mysql_test_data_timezone.sql",
	"fill_help_tables.sql",
}

var bsHeader = []string{
	"SET SESSION SQL_LOG_BIN = 0;",
	"CREATE DATABASE IF NOT EXISTS mysql;",
	"CREATE DATABASE IF NOT EXISTS test;",
	"USE mysql;",
}

var bsFooter = []string{
	"DELETE FROM mysql.user WHERE user = '';",
}

// appendLines will write the provided lines, newline-terminated, to
// the writer. If an error occurs when writing any of the lines, the
// writing will stop there and the error returned. This means that you
// have to make sure to clean up anything that could be partially
// written. In either case, the number of lines successfully written
// will be returned.
func appendLines(wr io.Writer, lines []string) (int, error) {
	count := 0
	for _, line := range lines {
		if _, err := fmt.Fprintln(wr, line); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// createBootstrap will create a bootstrap file for the server.
func (srv *Server) writeBootstrapFile(bs *os.File) error {
	log.Debugf("Creating bootstrap file %q\n", bs.Name())

	// Write the header to the bootstrap file
	if _, err := appendLines(bs, bsHeader); err != nil {
		return err
	}

	// Append bootstrap files from distribution
	for _, fname := range sqlFiles {
		fullname := filepath.Join(srv.Dist.Root, "share", fname)
		rd, err := os.Open(fullname)
		if err != nil {
			return err
		}
		_, err = io.Copy(bs, rd)
		rd.Close()
		if err != nil {
			return err
		}
	}

	// Append the footer lines to the bootstrap file
	if _, err := appendLines(bs, bsFooter); err != nil {
		return err
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
	log.Debug("Bootstrapping using", cmd.Args)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// fixDynamicFields will set the dynamic fields of the server if they
// are not already set. This is also used to handle updgrade of the
// configuration file when new fields are added.
func (srv *Server) fixDynamicFields() {
	if len(srv.BinPath) == 0 {
		srv.BinPath = srv.bin("mysqld")
	}
	if len(srv.LogPath) == 0 {
		srv.LogPath = srv.log("mysqld.err")
	}
	if len(srv.PidPath) == 0 {
		srv.PidPath = srv.run("mysqld.pid")
	}
	if len(srv.Socket) == 0 {
		srv.Socket = srv.run("mysqld.sock")
	}
}

// createServer will create and populate the server structure with the
// correct information.
func (stable *Stable) newServer(name string, dist *Dist) (*Server, error) {
	// Collect all the information
	baseDir := filepath.Join(stable.serverDir, name)
	dataDir := filepath.Join(baseDir, "data")
	cnfFile := filepath.Join(baseDir, "my.cnf")
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
		ServerId:   serverId,
		Options:    cnf.New(),
		Dist:       dist,
		User:       "root",
	}

	// Set up dynamic fields
	server.fixDynamicFields()

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
			"pid_file":  server.PidPath,
			"server_id": strconv.Itoa(serverId),
		},

		"mysql": map[string]string{
			"protocol": "tcp",
			"host":     server.Host,
			"port":     strconv.Itoa(server.Port),
			"prompt":   "'" + name + "> '",
		},
	}

	if dist.Version >= "5.1.6" {
		option["mysqld"]["log-output"] = "file"
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

// FindMatchingServers find all servers matching any of the patterns
// in the slice. The only possible returned error is
// filepath.ErrBadPattern, which is returned if any of the provided
// patterns is bad. Otherwise, an array of matcing servers is returned.
func (stable *Stable) FindMatchingServers(patterns []string) ([]*Server, error) {
	var servers []*Server

	for _, pattern := range patterns {
		for name, srv := range stable.Server {
			matched, err := filepath.Match(pattern, name)
			if err != nil {
				return nil, err
			} else if matched {
				servers = append(servers, srv)
			}
		}
	}

	return servers, nil
}

// setup will create all the necessary directories and files for a
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
		if err := os.Mkdir(dir, 0755); err != nil {
			return err
		}
	}

	cnfFile := filepath.Join(srv.BaseDir, "my.cnf")
	if fd, err := os.Create(cnfFile); err != nil {
		return err
	} else {
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

// Delete the server from the stable and remove all associated files.
func (stable *Stable) DelServer(srv *Server) error {
	if err := srv.teardown(); err != nil {
		return err
	}

	delete(stable.Server, srv.Name)
	return nil
}

var replRegex = regexp.MustCompile(`\{\w+\}`)

// fmtString will produce a formatted string from the server
// fields. This can probably be generalized to any interface type.
func (srv *Server) FormatString(format string) string {
	rsrv := reflect.Indirect(reflect.ValueOf(srv))
	res := replRegex.ReplaceAllFunc([]byte(format), func(m []byte) []byte {
		name := string(m[1 : len(m)-1])
		str := fmt.Sprintf("%v", rsrv.FieldByName(name).Interface())
		return []byte(str)
	})
	return string(res)
}

// Status will return the status of the server.
func (srv *Server) Status() Status {
	if _, err := os.Stat(srv.PidPath); err != nil {
		return SERVER_UNAVAIL
	} else {
		// TODO: add a ping-check to kill the server if it
		// does not reply properly
		return SERVER_RUNNING
	}
}

// Pid will get the server PID from the PID file, or return an error
// if the PID cannot be retrieved for some reason (such as that the
// file cannot be read, or does not exist).
func (srv *Server) Pid() (int, error) {
	if _, err := os.Stat(srv.PidPath); err != nil {
		return -1, fmt.Errorf("Server %q not running", srv.Name)
	}
	if file, err := os.Open(srv.PidPath); err != nil {
		return -1, fmt.Errorf("Open %q failed: %s", srv.Name, err)
	} else {
		var pid int
		if count, err := fmt.Fscanln(file, &pid); count < 1 {
			return -1, fmt.Errorf("Cannot read PID from file: %s", err)
		}
		return pid, nil
	}
}

// IsLocal will return true if the server is on the local host, false
// otherwise.
func (srv *Server) IsLocal() bool {
	// TODO check the host name of the machine?
	return srv.Host == "localhost" || strings.HasPrefix(srv.Host, "127.0.0")
}

func (s *Server) SocketDsn() string {
	return fmt.Sprintf("%v:%v@unix(%v)/%v", s.User, s.Password, s.Socket, s.database)
}

func (s *Server) TcpDsn() string {
	return fmt.Sprintf("%v:%v@tcp(%v:%v)/%v", s.User, s.Password, s.Host, s.Port, s.database)
}

// mysqlArgs return an array of default arguments for using a mysql
// client with the server.
func (srv *Server) mysqlArgs(args ...string) []string {
	argv := []string{
		fmt.Sprintf("-S%s", srv.Socket),
		fmt.Sprintf("-h%s", srv.Host),
		fmt.Sprintf("-P%d", srv.Port),
	}
	if len(srv.User) > 0 {
		argv = append(argv, fmt.Sprintf("-u%s", srv.User))
	}
	if len(srv.Password) > 0 {
		argv = append(argv, fmt.Sprintf("-p%s", srv.Password))
	}

	return append(argv, args...)
}

// Execute is used to execute a command using the mysql client for the
// server and return the result.
func (srv *Server) Execute(commands ...string) error {
	argv := srv.mysqlArgs("-e" + strings.Join(commands, ";"))
	cmd := exec.Command(srv.bin("mysql"), argv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Debugf("Executing %v", cmd.Args)
	return cmd.Run()
}

// Connect is used to connect a terminal to the server and run a
// prompt.
func (srv *Server) Connect(args ...string) error {
	argv := srv.mysqlArgs(args...)
	cmd := exec.Command(srv.bin("mysql"), argv...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Debugf("Executing %v", srv.Name, cmd.Args)
	return cmd.Run()
}
