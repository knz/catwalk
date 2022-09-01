package catwalk

import (
	"bytes"
	"fmt"
	"reflect"
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
	// TestUpdate is called for every unknown command
	// under "run" directives in the input file.
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

	result bytes.Buffer

	// Queued commands left for processing.
	cmds []tea.Cmd

	// Queued messages left for processing.
	msgs []tea.Msg

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
	//
	// The following directives are supported:
	//
	// - run: apply some state changes and view the result.
	//
	//   Supported directive options:
	//   - trace: produce a log of the intermediate test steps.
	//   - observe: what to observe after the state changes.
	//
	//     Supported values for observe:
	//     - view: the result of calling View().
	//     - gostruct: the result of printing the model with %#v.
	//     - debug: the result of calling the Debug() method (it needs to be defined)
	//     - msgs/cmds: print the residual tea.Cmd / tea.Msg input.
	//
	//   Supported input commands under "run":
	//   - type: enter some runes as tea.Key
	//   - key: enter a special key or combination as a tea.Key
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

func (d *driver) trace(traceEnabled bool, format string, args ...interface{}) {
	if traceEnabled {
		fmt.Fprintf(&d.result, "-- trace: "+format+"\n", args...)
	}
}

func (d *driver) processTeaCmds(trace bool) {
	if len(d.cmds) > 0 {
		d.trace(trace, "processing %d cmds", len(d.cmds))
	}
	// TODO(knz): handle timeouts.
	var inputs []tea.Cmd
	for {
		if len(d.cmds) >= 0 {
			inputs = append(make([]tea.Cmd, 0, len(d.cmds)+len(inputs)), inputs...)
			inputs = append(inputs, d.cmds...)
			d.cmds = nil
		}
		if len(inputs) == 0 {
			break
		}
		cmd := inputs[0]
		inputs = inputs[1:]
		msg := cmd()

		if msg != nil {
			rmsg := reflect.ValueOf(msg)
			if rmsg.CanConvert(cmdsType) {
				rcmds := rmsg.Convert(cmdsType)
				cmds := rcmds.Interface().([]tea.Cmd)
				d.trace(trace, "expanded %d commands", len(cmds))
				d.addCmds(cmds...)
				continue
			}
		}

		d.trace(trace, "translated cmd: %T", msg)
		d.addMsg(msg)
	}
}

var (
	cmdsType       = reflect.TypeOf([]tea.Cmd{})
	printType      = reflect.TypeOf(tea.Println("hello")())
	quitType       = reflect.TypeOf(tea.Quit())
	execType       = reflect.TypeOf(tea.ExecProcess(nil, nil)())
	hideCursorType = reflect.TypeOf(tea.HideCursor())
	enterAltType   = reflect.TypeOf(tea.EnterAltScreen())
	exitAltType    = reflect.TypeOf(tea.ExitAltScreen())
	mouseCellType  = reflect.TypeOf(tea.EnableMouseCellMotion())
	mouseAllType   = reflect.TypeOf(tea.EnableMouseAllMotion())
	mouseDisType   = reflect.TypeOf(tea.DisableMouse())
	szType         = reflect.TypeOf(tea.WindowSizeMsg{})
)

func (d *driver) processTeaMsgs(trace bool) {
	if len(d.msgs) > 0 {
		d.trace(trace, "processing %d messages", len(d.msgs))
	}
	for _, msg := range d.msgs {
		d.trace(trace, "msg %#v", msg)

		switch reflect.TypeOf(msg) {
		case printType:
			fmt.Fprintf(&d.result, "TEA PRINT: %v\n", msg)
		case szType:
			fmt.Fprintf(&d.result, "TEA WINDOW SIZE: %v\n", msg)
		case quitType:
			fmt.Fprintf(&d.result, "TEA QUIT\n")
		case execType:
			fmt.Fprintf(&d.result, "TEA EXEC\n")
		case hideCursorType:
			fmt.Fprintf(&d.result, "TEA HIDE CURSOR\n")
		case enterAltType:
			fmt.Fprintf(&d.result, "TEA ENTER ALT\n")
		case exitAltType:
			fmt.Fprintf(&d.result, "TEA EXIT ALT\n")
		case mouseCellType:
			fmt.Fprintf(&d.result, "TEA ENABLE MOUSE CELL MOTION\n")
		case mouseAllType:
			fmt.Fprintf(&d.result, "TEA ENABLE MOUSE MOTION ALL\n")
		case mouseDisType:
			fmt.Fprintf(&d.result, "TEA DISABLE MOUSE\n")
		default:
			newM, newCmd := d.m.Update(msg)
			d.m = newM
			d.addCmds(newCmd)
		}
	}
	d.msgs = d.msgs[:0]
}

func (d *driver) addCmds(cmds ...tea.Cmd) {
	for _, cmd := range cmds {
		if cmd == nil {
			continue
		}
		d.cmds = append(d.cmds, cmd)
	}
}

func (d *driver) addMsg(msg tea.Msg) {
	if msg == nil {
		return
	}
	d.msgs = append(d.msgs, msg)
}

func (d *driver) Close(t TB) {}

func (d *driver) RunOneTest(t TB, td *datadriven.TestData) string {
	// Save the input position.
	d.pos = td.Pos

	switch td.Cmd {
	case "run":
		d.result.Reset()

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

		traceEnabled := td.HasArg("trace")
		trace := func(format string, args ...interface{}) {
			d.trace(traceEnabled, format, args...)
		}

		doObserve := func() {
			for _, obs := range observe {
				o := d.Observe(t, obs)
				d.result.WriteString(o)
				// Terminate items with a newline if there's none yet.
				if d.result.Len() > 0 {
					if d.result.Bytes()[d.result.Len()-1] != '\n' {
						d.result.WriteByte('\n')
					}
				}
			}
		}

		// Process the initialization, if not done yet.
		if !d.startDone {
			if !d.disableAutoInit {
				trace("calling Init")
				d.addCmds(d.m.Init())
				d.processTeaCmds(traceEnabled)
			}

			if d.autoSize {
				msg := tea.WindowSizeMsg{Width: d.width, Height: d.height}
				d.addMsg(msg)
			}
			d.startDone = true
		}

		// Process the commands in the test's input.
		testInputCommands := strings.Split(td.Input, "\n")

		for _, testInputCmd := range testInputCommands {
			testInputCmd = strings.TrimSpace(testInputCmd)
			if testInputCmd == "" || strings.HasPrefix(testInputCmd, "#") {
				// Comment or emptyline.
				continue
			}

			trace("before %q", testInputCmd)

			// If the previous testInputCmd produced
			// some tea.Cmds, process them now.
			d.processTeaMsgs(traceEnabled)

			// Apply the new testInputCmd.
			args := strings.Split(testInputCmd, " ")
			testInputCmd = args[0]
			args = args[1:]
			cmd := d.ApplyTextCommand(t, testInputCmd, args...)
			d.addCmds(cmd)
			d.processTeaCmds(traceEnabled)

			if traceEnabled {
				trace("after %q", testInputCmd)
				doObserve()
			}
		}

		if traceEnabled {
			trace("before finish")
			doObserve()
		}
		// Last round of command execution.
		d.processTeaMsgs(traceEnabled)
		d.processTeaCmds(traceEnabled)
		d.processTeaMsgs(traceEnabled)

		trace("at end")
		doObserve()
		return d.result.String()

	default:
		t.Fatalf("%s: unrecognized test directive: %s", td.Pos, td.Cmd)
	}

	return "unreachable"
}

func (d *driver) Observe(t TB, what string) string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "-- %s:\n", what)
	switch what {
	case "msgs":
		fmt.Fprintf(&buf, "msg queue sz: %d\n", len(d.msgs))
		for i, msg := range d.msgs {
			t := reflect.TypeOf(msg)
			fmt.Fprintf(&buf, "%d:%s: %v\n", i, t, msg)
		}

	case "cmds":
		fmt.Fprintf(&buf, "command queue sz: %d\n", len(d.cmds))

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
			t.Fatalf("%s: model does not support a Debug() string method", d.pos)
		}
		buf.WriteString(md.Debug())

	case "gostruct":
		buf.WriteString(pretty.Sprint(d.m))

	default:
		t.Fatalf("%s: unsupported observation: %q", d.pos, what)
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
	alt := false

	switch cmd {
	case "resize":
		d.assertArgc(t, args, 2)
		w := d.getInt(t, args[0])
		h := d.getInt(t, args[1])
		msg := tea.WindowSizeMsg{Width: w, Height: h}
		d.addMsg(msg)

	case "key":
		d.assertArgc(t, args, 1)
		keyName := args[0]
		if strings.HasPrefix(keyName, "alt+") {
			alt = true
			keyName = strings.TrimPrefix(keyName, "alt+")
		}
		k, ok := allKeys[keyName]
		if !ok && len(keyName) != 1 {
			t.Fatalf("%s: unknown key: %s", d.pos, keyName)
		}
		if ok {
			k.Alt = alt
			msg := tea.KeyMsg(k)
			d.addMsg(msg)
			break
		}
		// Not a special key: it's runes.
		args[0] = keyName
		fallthrough

	case "type":
		var buf strings.Builder
		for i, arg := range args {
			if i > 0 {
				buf.WriteByte(' ')
			}
			buf.WriteString(arg)
		}
		for _, r := range buf.String() {
			d.addMsg(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{r}, Alt: alt}))
		}

	default:
		if d.upd != nil {
			t.Logf("%s: applying command %q via model updater", d.pos, cmd)
			newModel, cmd := d.upd.TestUpdate(t, d.m, cmd, args...)
			d.m = newModel
			return cmd
		} else {
			t.Fatalf("%s: unknown command %q, and no ModelUpdater defined", d.pos, cmd)
		}
	}

	return nil
}

var allKeys = func() map[string]tea.Key {
	result := make(map[string]tea.Key)
	for i := 0; ; i++ {
		k := tea.Key{Type: tea.KeyType(i)}
		keyName := k.String()
		// fmt.Println("found key:", keyName)
		if keyName == "" {
			break
		}
		result[keyName] = k
	}
	for i := -2; ; i-- {
		k := tea.Key{Type: tea.KeyType(i)}
		keyName := k.String()
		// fmt.Println("found key:", keyName)
		if keyName == "" {
			break
		}
		result[keyName] = k
	}
	return result
}()
