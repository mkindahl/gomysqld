// Copyright (c) 2014, Oracle and/or its affiliates. All rights reserved.

// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; version 2 of the License.

// This program is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program; if not, write to the Free Software
// Foundation, Inc., 51 Franklin St, Fifth Floor, Boston, MA 02110-1301
// USA

// Package to work with MySQL Server configuration files. It allow
// simple parsing of configuration files, updating it in-memory, and
// writing the updated version back to a file.
package cnf

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

var (
	ErrSectionPresent = errors.New("Section exists")
	ErrSectionMissing = errors.New("Section missing")
)

// Section is a section of the configuration file. Each section can
// contain mappings from options to values. The values are always
// stored as strings, but they can be converted on retrieval.
type Section struct {
	Header  []string
	options map[string]string
}

// Config is the configuration structure holding the sections and
// options.
type Config struct {
	Header  []string
	Section map[string]*Section
}

// New will create a new empty configuration structure.
func New() *Config {
	return &Config{
		Header:  make([]string, 0),
		Section: make(map[string]*Section),
	}
}

func (cnf *Config) AppendHeaderLine(section, line string) error {
	m, ok := cnf.Section[section]
	if !ok {
		return ErrSectionMissing
	}
	m.Header = append(m.Header, line)
	return nil
}

// AddSection will a new section to the configuration structure. If
// the section is already present, an error is returned.
func (cnf *Config) AddSection(section string) (*Section, error) {
	_, ok := cnf.Section[section]
	if ok {
		return nil, ErrSectionPresent
	}

	sec := &Section{
		Header:  make([]string, 0),
		options: make(map[string]string),
	}
	cnf.Section[section] = sec
	return sec, nil
}

// RemoveSection will remove a section from the configuration
// structure. If the section is missing from the structure, a
// ErrSectionMissing error is returned.
func (cnf *Config) RemoveSection(section string) error {
	_, exists := cnf.Section[section]
	if !exists {
		return fmt.Errorf("Section %q missing", section)
	}
	delete(cnf.Section, section)
	return nil
}

// GetString will return the value of an option in a section. If the
// section or option does not exist, an error is returned.
func (sec *Section) GetString(option string) string {
	return sec.options[option]
}

// Set will set the value of an option in a section. If the section
// did not exist prior to the call, the section will be created.
func (sec *Section) SetString(opt, val string) {
	sec.options[opt] = val
}

// ImportSection will import options into a single section.
func (sec *Section) Import(contents map[string]string) error {
	for opt, val := range contents {
		sec.SetString(opt, val)
	}
	return nil
}

// Import will import a map consisting of sections with settings. It
// is used to simplify the population of more extensive
// configurations.
func (cnf *Config) Import(sections map[string]map[string]string) error {
	for section, contents := range sections {
		sec, exists := cnf.Section[section]
		if !exists {
			if s, err := cnf.AddSection(section); err != nil {
				return err
			} else {
				sec = s
			}
		}

		if err := sec.Import(contents); err != nil {
			return err
		}
	}
	return nil
}

// Write will write the option structure to the given writer. If the
// structure was previously read from an options file, comments will
// not be written back.
func (cnf *Config) Write(wr io.Writer) error {
	for name, sec := range cnf.Section {
		fmt.Fprintf(wr, "\n\n")
		for _, line := range cnf.Header {
			fmt.Fprintf(wr, "# %s", line)
		}
		fmt.Fprintf(wr, "[%s]\n", name)
		for opt, val := range sec.options {
			fmt.Fprintln(wr, opt, "=", val)
		}
	}
	return nil
}

// trimLine will remove (and return) slices to the line (without
// leading and trailing whitespace) and comment (without leading and
// trailing whitespace).
func trimLine(line []byte) ([]byte, []byte) {
	if pos := bytes.IndexAny(line, ";#"); pos != -1 {
		result := bytes.TrimSpace(line[:pos])
		comment := bytes.TrimSpace(line[pos+1:])
		return result, comment
	} else {
		result := bytes.TrimSpace(line)
		return result, nil
	}
}

// swap will swap the guts of this configuration structure with
// another configuration structure. It is currently not atomic.
func (cnf *Config) swap(other *Config) {
	cnf.Header, other.Header = other.Header, cnf.Header
	cnf.Section, other.Section = other.Section, cnf.Section
}

// scanLogicalLines will find the end of a logical line, taking
// continuation lines into account. It will return lines without the
// training newline (if there is one) and with all continuation line
// breaks removed.
func scanLogicalLines(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// The token is created by extracting chunks of bytes from the
	// data and concatenating them. Each chunk is separated by
	// bytes that are not part of the final token.  The separators
	// are backslash-newline right now.
	var token []byte = make([]byte, 0, len(data))
	beg, end := 0, -1
	for i, v := range data {
		switch v {
		case '\\':
			// This chunk ends here. Backslash and newline
			// should not be included in the token.
			end = i
		case '\n':
			// If the end index was not set previously,
			// the chunk ends here.
			if end < 0 {
				end = i
			}

			// Extend the token with the new chunk
			// starting after the last seen newline (or
			// the beginning of the string) and ending
			// either before the backslash or before the
			// newline (if not preceeded by a backslash).
			token = append(token, data[beg:end]...)

			// If this newline was not a continuation,
			// return the token.
			if end == i {
				return i + 1, token, nil
			}

			// Start a new chunk by resetting the begin
			// and end indices.
			beg, end = i+1, -1
		}
	}

	if atEOF {
		// If we reached end of the stream without finding the
		// end of a token, return the tail as a separate
		// token.
		token = append(token, data[beg:]...)
		return len(data), token, nil
	} else {
		// If we have not yet reached end of the stream,
		// request more data.
		return 0, nil, nil
	}
}

// Read will read a configuration file from the provided reader rd and
// parse it as a MySQL configuration file. Each section may optionally
// be preceeded with a section comment which is an unbroken sequence
// of comment lines. The header will then be stored with the section
// and written back when the configuration file is written out.
func (cnf *Config) Read(rd io.Reader) error {
	scanner := bufio.NewScanner(rd)
	// MySQL do not accept continuation lines, but we do
	scanner.Split(scanLogicalLines)
	newCnf := New()
	section := ""
	headerLines := []string{}

	for scanner.Scan() {
		source := scanner.Text()
		line, comment := trimLine([]byte(source))

		switch {
		case len(bytes.TrimSpace([]byte(source))) == 0:
			// This was an empty line, so the header is cleared
			headerLines = []string{}

		case len(line) == 0:
			if comment != nil {
				headerLines = append(headerLines, string(comment))
			}

		case line[0] == '[' && line[len(line)-1] == ']':
			section = string(bytes.TrimSpace(line[1 : len(line)-1]))
			newCnf.AddSection(section)
			newCnf.Section[section].Header = headerLines
			headerLines = make([]string, 0)

		case line[0] == '!':
			panic("File inclusions not handled yet")

		default:
			i := bytes.IndexAny(line, ":=")
			option := bytes.TrimSpace(line[:i])
			value := bytes.TrimSpace(line[i+1:])
			newCnf.Section[section].SetString(string(option), string(value))
		}
	}

	cnf.swap(newCnf)
	return nil
}
