package cnf

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
)

type Result struct {
	line    string
	comment []byte
}

func TestTrimLine(t *testing.T) {
	samples := map[string]Result{
		"":                    Result{"", nil},
		"  ":                  Result{"", nil},
		"  foo  ":             Result{"foo", nil},
		"foo  ":               Result{"foo", nil},
		"  foo":               Result{"foo", nil},
		"foo":                 Result{"foo", nil},
		"#foo":                Result{"", []byte("foo")},
		"x= 12 #just a test":  Result{"x= 12", []byte("just a test")},
		"x= 42 ;just a test":  Result{"x= 42", []byte("just a test")},
		"  x=12#just a test":  Result{"x=12", []byte("just a test")},
		"  x=12;just a test":  Result{"x=12", []byte("just a test")},
		"  x=12;#just a test": Result{"x=12", []byte("#just a test")},
	}

	for sample, expected := range samples {
		line, comment := trimLine([]byte(sample))
		if string(line) != expected.line {
			t.Errorf("Expected line %q, got %q\n", expected.line, line)
		}

		// Check the comment. Either both should be nil, or
		// they should contain the same bytes.
		if (comment != nil) != (expected.comment != nil) {
			t.Errorf("Expected %v, got %v\n", expected.comment, comment)
		} else if string(comment) != string(expected.comment) {
			t.Errorf("Expected line %q, got %q\n", expected.comment, comment)
		}
	}

}

func TestScanLogicalLines(t *testing.T) {
	samples := `
foo = bar
foo \
= bar
foo \
= \
bar
fo\
o\
 = bar`

	scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(samples)))
	scanner.Split(scanLogicalLines)
	expected := "foo = bar"
	count := 0
	for scanner.Scan() {
		if scanner.Text() != expected {
			t.Errorf("Expected %q, got %q", expected, scanner.Text())
		} else {
			count++
		}
	}

	if scanner.Err() != nil {
		t.Errorf("Expected no error, got '%v'", scanner.Err())
	}

	if count != 4 {
		t.Errorf("Expected %d matches, but was %d", 4, count)
	}

	scanner = bufio.NewScanner(strings.NewReader(`
# This is just a comment
[first]
alpha: 1
beta = 2         # Test

; This is the second section
[second]
gamma : 3;Another test
`))
	scanner.Split(scanLogicalLines)

	lines := []string{
		"",
		"# This is just a comment",
		"[first]",
		"alpha: 1",
		"beta = 2         # Test",
		"",
		"; This is the second section",
		"[second]",
		"gamma : 3;Another test",
	}

	i := 0
	for ; scanner.Scan(); i++ {
		if scanner.Text() != lines[i] {
			t.Errorf("Expected %q, got %q", lines[i], scanner.Text())
		}
	}

	if scanner.Err() != nil {
		t.Errorf("Expected no error, got %q", scanner.Err())
	}

	if i < len(lines) {
		t.Errorf("Expected to read %d lines, but read %d", len(lines), i)
	}
}

func TestSections(t *testing.T) {
	section := "my_section"

	config := New()

	if _, err := config.AddSection(section); err != nil {
		t.Errorf("Got unexpected error %q", err)
	}

	if _, err := config.AddSection(section); err == nil {
		t.Errorf("Expected error, got none")
	}

	if _, exists := config.Section[section]; !exists {
		t.Errorf("Section %q missing", section)
	}

	if err := config.RemoveSection(section); err != nil {
		t.Errorf("Got unexpected error %q", err)
	}

	if _, exists := config.Section[section]; exists {
		t.Errorf("Section %q not removed", section)
	}

	if err := config.RemoveSection(section); err == nil {
		t.Errorf("Expected error, got none")
	}
}

func TestOptions(t *testing.T) {
	section, option, value, expect := "my_section", "my_option", "test", "test"

	config := New()

	config.AddSection(section)
	config.Section[section].SetString(option, value)
	val := config.Section[section].GetString(option)
	if val != expect {
		t.Errorf("Expected %q for option %q in section %q, found %q", expect, option, section, value)
	}
}

func TestImport(t *testing.T) {
	cnf := New()

	sample := map[string]map[string]string{
		"first": {
			"alpha": "one",
			"beta":  "two",
		},
		"second": {
			"gamma": "three",
		},
	}

	cnf.Import(sample)

	for sec, contents := range sample {
		for opt, val := range contents {
			res := cnf.Section[sec].GetString(opt)
			if res != val {
				t.Errorf("Expected %q in section %q to have %q, had %q", opt, sec, val, res)
			}
		}
	}
}

func TestWriteRead(t *testing.T) {
	cnf := New()

	sample := map[string]map[string]string{
		"first": {
			"alpha": "one",
			"beta":  "two",
		},
		"second": {
			"gamma": "three",
		},
	}

	cnf.Import(sample)

	filename := "test.cnf"

	if fd, err := os.Create(filename); err != nil {
		t.Fatalf("Unable to create %q: %s", filename, err)
	} else {
		cnf.Write(fd)
		fd.Close()
	}

	cnf.Section["second"].SetString("delta", "four") // Should not exist after reloading

	if fd, err := os.Open(filename); err != nil {
		t.Fatalf("Unable to open file %q", filename)
	} else {
		cnf.Read(fd)
	}

	for sec, contents := range sample {
		for opt, val := range contents {
			res := cnf.Section[sec].GetString(opt)
			if res != val {
				t.Errorf("Expected %q in section %q to have %q, had %q", opt, sec, val, res)
			}
		}
	}
}

func TestReadWrite(t *testing.T) {
	cnf := New()
	cnf.Read(strings.NewReader(`
# This is just a comment
[first]
alpha: 1
beta = 2         # Test

; This is the second section
[second]
gamma : 3;Another test
`))

}

func TestRead1(t *testing.T) {
	cnf := New()
	cnf.Read(strings.NewReader(`
# This is a configuration file header

# This is a header for the first section
[first]
alpha: 1
beta = 2         # Test

# This is just an interim comment, it should not be attached to any
# section

; This is the second section
; with a multi-line header
[second]
gamma : 3;Another test
`))

	expected := map[string]map[string]string{
		"first": {
			"alpha": "1",
			"beta":  "2",
		},
		"second": {
			"gamma": "3",
		},
	}

	for sec, contents := range expected {
		for opt, val := range contents {
			res := cnf.Section[sec].GetString(opt)
			if res != val {
				t.Errorf("Expected %q for option %q in section %q, got %q", val, opt, sec, res)
			}
		}
	}

	comments := map[string]([]string){
		"first": {
			"This is a header for the first section",
		},
		"second": {
			"This is the second section",
			"with a multi-line header",
		},
	}

	for name, sec := range cnf.Section {
		strHeader := fmt.Sprintf("%v", sec.Header)
		strExpect := fmt.Sprintf("%v", comments[name])
		if strHeader != strExpect {
			t.Errorf("Expected header %v for section %q, got %v", strExpect, name, strHeader)
		}
	}

}
