package catwalk

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/datadriven"
	"github.com/kr/pretty"
)

// ModelUpdater is an optional object passed alongside
// a tea.Model in RunModel() which can apply state
// change commands as input to a test.
type ModelUpdater interface {
	// TestUpdate is called for every unknown directive
	// in the input.
	TestUpdate(t TB, m tea.Model, cmd string, args ...string) (tea.Model, tea.Cmd)
}

// RunModel runs the tests contained in the file pointed to by 'path'
// on the model m.
// To apply RunModel on all the test files in a directory,
// use datadriven.Walk.
func RunModel(t *testing.T, path string, m tea.Model, upd ModelUpdater) {
	driver := NewDriver(m, upd)
	defer driver.Close(t)

	driver.Start(t)

	datadriven.RunTest(t, path, func(t *testing.T, d *datadriven.TestData) string {
		return driver.RunOneTest(t, d)
	})
}

// driver represents the test driver.
type driver struct {
	m   tea.Model
	upd ModelUpdater

	// pos is the position in the input data file.
	// Used to produce error messages etc.
	pos string
}

// TB is a shim interface for testing.T / testing.B.
type TB interface {
	Fatal(...interface{})
	Fatalf(string, ...interface{})
	Logf(string, ...interface{})
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
func NewDriver(m tea.Model, upd ModelUpdater) Driver {
	return &driver{m: m, upd: upd}
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
		// Observations: check if there's an observe=() key
		// on the first test input line. If not, just observe the view.
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

		// Construct the expected result.
		var result bytes.Buffer
		for _, obs := range observe {
			o := d.Observe(t, obs)
			// Make newlines visible.
			o = strings.ReplaceAll(o, "\n", "␤\n")
			// Add a newline if there was none at the end.
			if len(o) == 0 || o[len(o)-1] != '\n' {
				o += "🛇\n"
			}

			result.WriteString(o)
		}
		return result.String()

	default:
		t.Fatalf("%s: unrecognized command: %s", td.Pos, td.Cmd)
	}

	return "unreachable"
}

func (d *driver) Observe(t TB, what string) string {
	switch what {
	case "view":
		return d.m.View()

	case "debug":
		type dbg interface{ Debug() string }
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

	return "unreachable"
}

func (d *driver) assertArgc(t TB, args []string, expected int) {
	if len(args) != expected {
		t.Fatalf("%s: expected %d args, got %d", d.pos, expected, len(args))
	}
}

func (d *driver) getInt(t TB, v string) int {
	i, err := strconv.Atoi(v)
	if err != nil {
		t.Fatalf("%s: %v", d.pos, err)
	}
	return i
}

func (d *driver) ApplyTextCommand(t TB, cmd string, args ...string) {
	switch cmd {
	case "resize":
		d.assertArgc(t, args, 2)
		w := d.getInt(t, args[0])
		h := d.getInt(t, args[1])
		msg := tea.WindowSizeMsg{Width: w, Height: h}
		newModel, cmd := d.m.Update(msg)
		_ = cmd
		// FIXME: use cmd
		d.m = newModel

	case "type":
		var buf strings.Builder
		for i, arg := range args {
			if i > 0 {
				buf.WriteByte(' ')
			}
			buf.WriteString(arg)
		}
		messages := make([]tea.Msg, 0, buf.Len())
		for _, r := range buf.String() {
			messages = append(messages, tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{r}}))
		}
		for _, msg := range messages {
			newModel, cmd := d.m.Update(msg)
			// FIXME: use cmd
			_ = cmd
			d.m = newModel
		}

	default:
		if d.upd != nil {
			t.Logf("%s: applying command %q via model updater", d.pos, cmd)
			newModel, cmd := d.upd.TestUpdate(t, d.m, cmd, args...)
			// FIXME: use cmd
			_ = cmd
			d.m = newModel
		} else {
			t.Fatalf("%s: unknown command %q, and no ModelUpdater defined", d.pos, cmd)
		}
	}
}
