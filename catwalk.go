package catwalk

import (
	"bytes"
	"fmt"
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

// Option is the type of an option which can be specified
// with RunModel or NewDriver.
type Option func(*driver)

// WithUpdater adds the specified model updater to the test.
func WithUpdater(upd ModelUpdater) Option {
	return func(d *driver) {
		d.upd = upd
	}
}

// WithAutoInitDisabled tells the test to not automatically
// initialize the model (via the Init method) upon first use.
func WithAutoInitDisabled() Option {
	return func(d *driver) {
		d.disableAutoInit = true
	}
}

// WithWindowSize tells the test driver to issue a tea.WindowSizeMsg
// as the first event after initialization.
func WithWindowSize(width, height int) Option {
	return func(d *driver) {
		d.autoSize = true
		d.width = width
		d.height = height
	}
}

// RunModel runs the tests contained in the file pointed to by 'path'
// on the model m, using a fresh driver initialize via NewDriver and
// the specified options.
//
// To apply RunModel on all the test files in a directory,
// use datadriven.Walk.
func RunModel(t *testing.T, path string, m tea.Model, opts ...Option) {
	d := NewDriver(m, opts...)
	defer d.Close(t)

	datadriven.RunTest(t, path, func(t *testing.T, td *datadriven.TestData) string {
		return d.RunOneTest(t, td)
	})
}

// driver represents the test driver.
type driver struct {
	m tea.Model

	// Queued commands left for processing.
	cmds []tea.Cmd

	// Test model updater (optional).
	upd ModelUpdater

	startDone bool

	// Don't call m.Init() on start.
	disableAutoInit bool

	// Send a WindowSizeMsg on start.
	autoSize bool
	width    int
	height   int

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
	// Close stops the model at the end.
	Close(t TB)
	// ApplyTextCommand applies the given textual command to the model.
	// It may return an extra tea.Cmd to process by the test.
	ApplyTextCommand(t TB, cmd string, args ...string) tea.Cmd
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
func NewDriver(m tea.Model, opts ...Option) Driver {
	d := &driver{
		m: m,
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

func (d *driver) processTeaCmds(trace bool) {
	// TODO
}

func (d *driver) addCmd(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	d.cmds = append(d.cmds, cmd)
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

		var result bytes.Buffer
		traceEnabled := td.HasArg("trace")
		trace := func(format string, args ...interface{}) {
			if traceEnabled {
				fmt.Fprintf(&result, "-- trace: "+format+"\n", args...)
			}
		}

		doObserve := func() {
			for _, obs := range observe {
				o := d.Observe(t, obs)
				result.WriteString(o)
				// Terminate items with a newline if there's none yet.
				if result.Len() > 0 {
					if result.Bytes()[result.Len()-1] != '\n' {
						result.WriteByte('\n')
					}
				}
			}
		}

		// Process the initialization, if not done yet.
		if !d.startDone {
			if !d.disableAutoInit {
				trace("calling Init")
				d.addCmd(d.m.Init())
			}
			if d.autoSize {
				msg := tea.WindowSizeMsg{Width: d.width, Height: d.height}
				trace("calling Update with initial %#v", msg)
				m, newCmd := d.m.Update(msg)
				d.m = m
				d.addCmd(newCmd)
			}
			d.startDone = true
		}

		// Process queued commands, if any.
		d.processTeaCmds(traceEnabled)

		// Process the commands in the test's input.
		testInputCommands := strings.Split(td.Input, "\n")

		for _, testInputCmd := range testInputCommands {
			testInputCmd = strings.TrimSpace(testInputCmd)
			if testInputCmd == "" || strings.HasPrefix(testInputCmd, "#") {
				// Comment or emptyline.
				continue
			}

			trace("before %q", testInputCmd)

			// If the previou testInputCmd produced
			// some tea.Cmd, process them now.
			d.processTeaCmds(traceEnabled)

			// Apply the new testInputCmd.
			args := strings.Split(testInputCmd, " ")
			testInputCmd = args[0]
			args = args[1:]
			cmd := d.ApplyTextCommand(t, testInputCmd, args...)
			d.addCmd(cmd)
		}

		// Final observation.
		trace("at end")
		doObserve()
		return result.String()

	default:
		t.Fatalf("%s: unrecognized test directive: %s", td.Pos, td.Cmd)
	}

	return "unreachable"
}

func (d *driver) Observe(t TB, what string) string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "-- %s:\n", what)
	switch what {
	case "cmds":
		fmt.Fprintf(&buf, "command queue sz: %d\n", len(d.cmds))
		for i, cmd := range d.cmds {
			fmt.Fprintf(&buf, "%d:%T: %v\n", i, cmd, cmd)
		}

	case "view":
		o := d.m.View()
		// Make newlines visible.
		o = strings.ReplaceAll(o, "\n", "␤\n")
		// Add a "no newline at end" marker if there was no newline at the end.
		if len(o) == 0 || o[len(o)-1] != '\n' {
			o += "🛇"
		}
		buf.WriteString(o)

	case "debug":
		type dbg interface{ Debug() string }
		md, ok := d.m.(dbg)
		if !ok {
			t.Fatalf("%s: model does not support a Debug() string method")
		}
		buf.WriteString(md.Debug())

	case "gostruct":
		buf.WriteString(pretty.Sprint(&buf, d.m))

	default:
		t.Fatalf("%s: unsupported observation: %q", what)
	}

	return buf.String()
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

func (d *driver) ApplyTextCommand(t TB, cmd string, args ...string) tea.Cmd {
	switch cmd {
	case "resize":
		d.assertArgc(t, args, 2)
		w := d.getInt(t, args[0])
		h := d.getInt(t, args[1])
		msg := tea.WindowSizeMsg{Width: w, Height: h}
		newModel, cmd := d.m.Update(msg)
		d.m = newModel
		return cmd

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
		var cmd tea.Cmd
		for _, msg := range messages {
			newModel, newCmd := d.m.Update(msg)
			d.m = newModel
			cmd = tea.Batch(cmd, newCmd)
		}
		return cmd

	default:
		if d.upd != nil {
			t.Logf("%s: applying command %q via model updater", d.pos, cmd)
			newModel, cmd := d.upd.TestUpdate(t, d.m, cmd, args...)
			// FIXME: use cmd
			d.m = newModel
			return cmd
		} else {
			t.Fatalf("%s: unknown command %q, and no ModelUpdater defined", d.pos, cmd)
		}
	}

	return nil // unreachable
}
