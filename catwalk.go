package catwalk

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/datadriven"
	"github.com/kr/pretty"
)

// RunModel runs the tests contained in the file pointed to by 'path'
// on the model m.
// To apply RunModel on all the test files in a directory,
// use datadriven.Walk.
func RunModel(t *testing.T, path string, m tea.Model) {
	driver := NewDriver(m)
	defer driver.Close(t)

	driver.Start(t)

	datadriven.RunTest(t, path, func(t *testing.T, d *datadriven.TestData) string {
		return driver.RunOneTest(t, d)
	})
}

// driver represents the test driver.
type driver struct {
	m tea.Model

	// pos is the position in the input data file.
	// Used to produce error messages etc.
	pos string
}

// TB is a shim interface for testing.T / testing.B.
type TB interface {
	Fatal(...interface{})
	Fatalf(string, ...interface{})
}

// Driver is the externally-visible interface for a test driver.
type Driver interface {
	// Start initializes the model and prepares it for testing.
	Start(t TB)
	// Close stops the model at the end.
	Close(t TB)
	// ApplyTextCommand applies the given textual command to the model.
	ApplyTextCommand(t TB, cmd string, args ...string)
	// Observe observes the given component of the model.
	// Supported values:
	// - view: call View()
	// - gostruct: print with %#v
	// - debug: call Debug()
	Observe(t TB, what string) string
	// RunOneTest runs one step of a test file.
	RunOneTest(t TB, d *datadriven.TestData) string
}

// NewDriver creates a test driver for the given model.
func NewDriver(m tea.Model) Driver {
	return &driver{m: m}
}

func (d *driver) Start(t TB) {
	// FIXME: use cmd
	_ = d.m.Init()
}

func (d *driver) Close(t TB) {}

func (d *driver) RunOneTest(t TB, td *datadriven.TestData) string {
	// Save the input position.
	d.pos = td.Pos

	switch td.Cmd {
	case "run":
		// Process the commands in the input.
		commands := strings.Split(td.Input, "\n")
		for _, command := range commands {
			command = strings.TrimSpace(command)
			if command == "" || strings.HasPrefix(command, "#") {
				// Comment or emptyline.
				continue
			}
			args := strings.Split(command, " ")
			command = args[0]
			args = args[1:]
			d.ApplyTextCommand(t, command, args...)
		}

	default:
		t.Fatalf("%s: unrecognized command: %s", td.Pos, td.Cmd)
	}

	// Observations: check if there's an observe=() key
	// on the test input. If not, just observe the view.
	var observe []string
	seen := false
	for i := range td.CmdArgs {
		if td.CmdArgs[i].Key == "observe" {
			observe = td.CmdArgs[i].Vals
			seen = true
			break
		}
	}
	if !seen {
		observe = []string{"view"}
	}
	// Construct the expected result.
	var result bytes.Buffer
	for _, obs := range observe {
		result.WriteString(d.Observe(t, obs))
		// Add a newline if there was none at the end.
		b := result.Bytes()
		if len(b) == 0 || b[len(b)-1] != '\n' {
			result.WriteString("🛇\n")
		}
	}
	return result.String()
}

func (d *driver) Observe(t TB, what string) {
	switch what {
	case "view":
		return d.m.View()

	case "debug":
		type dbg interface{ Debug() String }
		md, ok := d.m.(dbg)
		if !ok {
			t.Fatalf("%s: model does not support a Debug() string method")
		}
		return md.Debug()

	case "gostruct":
		return pretty.Sprint(d.m)

	default:
		t.Fatalf("%s: unsupported observation: %q", what)
	}
}

func (d *driver) ApplyTextCommand(t TB, cmd string, args ...string) {
	switch cmd {
	case "resize":

	}
}
