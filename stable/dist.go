/*

Package `mysqld/stable` is used to create and destroy new servers for
testing, experimentation, and benchmarking.

To create servers, it is first necessary to create a stable that will
contain the distributions and servers for the experiment. The stable is
just a directory where information will be stored. Once a stable is
either created to loaded, you can add distributions. The distributions
contain the actual server code and and added by using a binary
distribution either as a tar file, a zip file, or a directory.

*/
package stable

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Dist hold information about distribution.
type Dist struct {
	Root                         string
	Name, Version, ServerVersion string
	stable                       *Stable
	defaultPort                  int
}

// validateTar check a tar archive (compressed or not) to ensure that
// it has all the components needed to bootstrap a slave.
func (dt *Dist) unpackTar(root, path string) error {
	base := filepath.Base(path)
	dt.Name = strings.TrimSuffix(base, ".tar.gz")
	dt.Root = filepath.Join(root, dt.Name)

	// Extract the contents of the library
	cmd := exec.Command("tar", "xzf", path, "-C", root)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (dt *Dist) unpackZip(root, path string) error {
	base := filepath.Base(path)
	dt.Name = strings.TrimSuffix(base, ".zip")
	dt.Root = filepath.Join(root, dt.Name)

	// Extract the contents of the library
	cmd := exec.Command("unzip", "-qq", "-d", dt.Name, path)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

type DistType int

const (
	UNKNOWN_PATH = iota
	TGZ_PATH
	TAR_PATH
	ZIP_PATH
	DIR_PATH
)

// pathType will check the name of the of the file and based on this
// decide what type of distribution it is.
func pathType(path string) DistType {
	base := filepath.Base(path)
	if isTgz, _ := filepath.Match("*.tar.gz", base); isTgz {
		return TGZ_PATH
	} else if isTar, _ := filepath.Match("*.tar", base); isTar {
		return TAR_PATH
	} else if isZip, _ := filepath.Match("*.zip", base); isZip {
		return ZIP_PATH
	} else if finfo, err := os.Stat(path); err == nil && finfo.IsDir() {
		return DIR_PATH
	}
	return UNKNOWN_PATH
}

// unpackFrom will ensure that the distribution is unpacked and
// installed in the distribution tree under the root directory for
// distributions. If this function finishes successfully, nil is
// returned, otherwise, an error is returned.
func (dt *Dist) unpackDist(root, path string) error {
	log.Printf("Unpacking distribution %s into %s\n", path, root)
	switch pathType(path) {
	case TGZ_PATH:
		return dt.unpackTar(root, path)
	case ZIP_PATH:
		return dt.unpackZip(root, path)
	case DIR_PATH:
		dt.Name = filepath.Base(path)
		dt.Root = filepath.Join(root, dt.Name)
		os.Symlink(path, dt.Name)
		return nil
	default:
		return ErrInvalidDist
	}
}

const (
	definePattern = `^#\s*define\s+(\w+)\s+(.*)`
)

var (
	defineRegex = regexp.MustCompile(definePattern)
)

var includeFiles = []string{
	"include/mysql_version.h",
}

func (dt *Dist) scanVersionFile(src io.Reader) (outerr error) {
	// Scan the file to find the version. Right now, only the
	// server version is extracted but there could be other
	// information in the future that we should fetch here.
	scanner := bufio.NewScanner(src)
	outerr = ErrVersionNotFound
	for scanner.Scan() {
		match := defineRegex.FindStringSubmatch(scanner.Text())
		if match != nil {
			switch match[1] {
			case "MYSQL_SERVER_VERSION":
				dt.Version = strings.Trim(match[2], `" `)
				outerr = nil
			case "MYSQL_PORT":
				port, err := strconv.ParseInt(strings.Trim(match[2], `" `), 10, 0)
				if err != nil {
					return err
				}
				dt.defaultPort = int(port)
			}
		}
	}
	return
}

// checkDistFiles will check that all expected files exists in the
// distribution.
func (dt *Dist) checkDistFiles(files []string) error {
	for _, path := range files {
		_, err := os.Stat(filepath.Join(dt.Root, path))
		if err != nil {
			return err
		}
	}
	return nil
}

// Example: "mysqld  Ver 5.5.32-0ubuntu0.12.04.1-log for debian-linux-gnu on i686 ((Ubuntu))"
func (dt *Dist) parseVersionString(version string) {
	re := regexp.MustCompile(`^\S+\s+Ver\s+(\d+\.\d+\.\d+\S*)\s+for\s+(\S+)`)
	if match := re.FindStringSubmatch(version); match != nil {
		dt.ServerVersion = match[1]
	}
}

// readVersionFile will extract information from the version file of
// an unpacked distribution.
func (dt *Dist) readVersionFile() error {
	// Open the file containing server version information
	verFn := filepath.Join(dt.Root, "include", "mysql_version.h")
	fi, err := os.Open(verFn)
	if err != nil {
		return ErrInvalidDist
	}

	return dt.scanVersionFile(fi)
}

func (dt *Dist) readServerInfo() error {
	mysqld := filepath.Join(dt.Root, "bin", "mysqld")
	if ver, err := exec.Command(mysqld, "--version").Output(); err != nil {
		return err
	} else {
		dt.parseVersionString(string(ver))
	}
	return nil
}

// newDist is used to create a new distribution memory structure.
func (stable *Stable) newDist() (*Dist, error) {
	dist := &Dist{
		stable:      stable,
		defaultPort: 3306,
	}
	return dist, nil
}

func (dt *Dist) setup(stable *Stable, path string) error {
	// Unpack the distribution into the stable.
	if err := dt.unpackDist(stable.distDir, path); err != nil {
		return err
	}

	// Check that all files needed exists
	if err := dt.checkDistFiles(sqlFiles); err != nil {
		return err
	}
	if err := dt.checkDistFiles(includeFiles); err != nil {
		return err
	}

	// Extract information from the distribution.
	if err := dt.readVersionFile(); err != nil {
		return err
	}

	if err := dt.readServerInfo(); err != nil {
		return err
	}

	return nil
}

// AddDist is used to create a new distribution from some source
// given by the path. The source have to be a binary distribution, but
// it can be either a tar file, an unpacked directory, or a zip file
// with the binary distribution.  If it is a archive of any form, it
// is unpacked into the stable, but if it is a directory, a soft link
// is created in the stable to the real directory.
func (stable *Stable) AddDist(path string) (*Dist, error) {
	dt, err := stable.newDist()
	if err != nil {
		return nil, err
	}

	// Try to set up the distribution. If it is not possible due
	// to some error, the distribution is removed and the error
	// reported.
	if err := dt.setup(stable, path); err != nil {
		if len(dt.Root) == 0 {
			os.RemoveAll(dt.Root)
		}
		return nil, err
	}

	stable.Distro[dt.Name] = dt
	return dt, nil
}

// DelDist will remove the distribution from the stable, including all
// servers using the distribution.
func (stable *Stable) DelDistByName(name string) error {
	dist, exists := stable.Distro[name]
	if !exists {
		return fmt.Errorf("No distribution named %q exists", name)
	}
	for _, srv := range stable.Server {
		if srv.Dist == dist {
			stable.DelServer(srv)
		}
	}

	delete(stable.Distro, dist.Name)
	return nil
}
