// This package contain a library for creating scratch servers for,
// e.g., testing purposes. It support creating servers based on
// different distributions, meaning that you can, for example, test
// that statements work with several versions of servers.

package stable

import (
	"encoding/json"
	"io/ioutil"
	"mysqld/log"
	"os"
	"path/filepath"
	"syscall"
)

const (
	// STABLE_DIR is the default name for the stable directory
	STABLE_DIR  = ".stable"
	CONFIG_FILE = "config.json"
)

type Stable struct {
	// Root is the directory where the stable is positioned.
	Root string

	// Dist is a map from distribution names to distributions. The
	// name is taken from the output of mysqld --version
	Distro map[string]*Dist
	Server map[string]*Server

	NextPort, NextServerId int

	distDir, serverDir, tmpDir string
}

// nextPort allocate a new port number for a server
func (stable *Stable) fetchPortNumber() int {
	stable.NextPort++
	return stable.NextPort - 1
}

// fetchServerId allocate a new server identifier for a server
func (stable *Stable) fetchServerId() int {
	stable.NextServerId++
	return stable.NextServerId - 1
}

// absPath turn a relative path into an absolute path, but leaves
// absolute paths untouched. If the path is relative, the current
// working directory is used as origin for the relative location.
func absPath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		root, err := os.Getwd()
		if err != nil {
			return "", err
		}
		path = filepath.Join(root, path)
	}
	return path, nil
}

// configFile return the name of the configuration file for the MySQL
// stable.
func (stable *Stable) configFile() string {
	return filepath.Join(stable.Root, CONFIG_FILE)
}

// ReadConfig read a configuration file and populate the structure.
func (stable *Stable) ReadConfig() error {
	path := stable.configFile()
	rd, err := os.Open(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(rd)
	if err := decoder.Decode(stable); err != nil {
		return err
	}

	// Set the dynamic fields of the server after reading the
	// configuration file, in case new fields were added.
	for _, srv := range stable.Server {
		srv.fixDynamicFields()
	}

	return nil
}

// WriteConfig write the configuration to the configuration file.
func (stable *Stable) WriteConfig() error {
	path := stable.configFile()
	wr, err := ioutil.TempFile(filepath.Dir(path), "config")
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(wr)
	err = encoder.Encode(stable)
	wr.Close()
	if err != nil {
		os.Remove(wr.Name())
		return err
	}
	if err := os.Rename(wr.Name(), path); err != nil {
		os.Remove(wr.Name())
		return err
	}
	return nil
}

// newStable allocate the stable structure and fill in the fields
// correctly. It accepts a path as parameter which will be used as
// root for the stable.
func newStable(path string) (*Stable, error) {
	absPath, err := absPath(filepath.Join(path, STABLE_DIR))
	if err != nil {
		return nil, err
	}

	stable := &Stable{
		Root:         absPath,
		Distro:       make(map[string]*Dist),
		Server:       make(map[string]*Server),
		NextPort:     12000,
		NextServerId: 1,
		distDir:      filepath.Join(absPath, "dist"),
		serverDir:    filepath.Join(absPath, "server"),
		tmpDir:       filepath.Join(absPath, "tmp"),
	}

	return stable, nil
}

// create will create the necessary files and directories to set up
// the stable.  Distributions are stored under the "dist" directory,
// where there is one directory for each distribution.  Server data is
// stored under the "server" directory, where there is one directory
// for each server.
func (stable *Stable) setup() error {
	log.Debugf("Creating files and directories for stable in %q", stable.Root)

	// Create the stable directory
	if err := os.Mkdir(stable.Root, 0755); err != nil {
		errno := err.(*os.PathError).Err.(syscall.Errno)
		if errno == syscall.EEXIST {
			return ErrStableExists
		} else {
			return err
		}
	}

	dirs := []string{
		stable.distDir,
		stable.serverDir,
		stable.tmpDir,
	}
	for _, dir := range dirs {
		if err := os.Mkdir(dir, 0755); err != nil {
			os.RemoveAll(stable.Root)
			return err
		}
	}

	return nil
}

func (stable *Stable) teardown() error {
	log.Infof("Destroying stable in %q\n", stable.Root)
	return os.RemoveAll(stable.Root)
}

// CreateStable creates a new stable where distributions and servers
// can be created.
func CreateStable(path string) (*Stable, error) {
	stable, err := newStable(path)
	if err != nil {
		return nil, err
	}

	log.Infof("Creating stable in %q", stable.Root)

	if err := stable.setup(); err != nil {
		return nil, err
	}

	if err := stable.WriteConfig(); err != nil {
		return nil, err
	}

	return stable, nil
}

// Open is used to open an existing stable at the given path. If
// successful, a new stable is returned.
func OpenStable(path string) (*Stable, error) {
	stable, err := newStable(path)
	log.Infof("Opening stable in %q", stable.Root)
	if err != nil {
		return nil, err
	}
	if err := stable.ReadConfig(); err != nil {
		return nil, err
	}
	return stable, nil
}

func (stable *Stable) Destroy() error {
	log.Infof("Destroying stable in %q", stable.Root)
	return stable.teardown()
}
