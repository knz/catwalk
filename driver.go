package catwalk

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/datadriven"
	"github.com/kr/pretty"
)

// driver represents the test driver.
type driver struct {
	ctx    context.Context
	cancel func()

	m tea.Model

	result bytes.Buffer

	// Queued commands left for processing.
	cmds []tea.Cmd

	// cmdTimeout is how long to wait for a tea.Cmd
	// to return a tea.Msg.
	cmdTimeout time.Duration

	// Queued messages left for processing.
	msgs []tea.Msg

	// Test observers.
	observers map[string]Observer

	// Test model updater (optional).
	upd Updater

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

const defaultCmdTimeout time.Duration = 20 * time.Millisecond

// NewDriver creates a test driver for the given model.
func NewDriver(m tea.Model, opts ...Option) Driver {
	ctx, cancel := context.WithCancel(context.Background())
	d := &driver{
		ctx:    ctx,
		cancel: cancel,

		m:          m,
		cmdTimeout: defaultCmdTimeout,
		observers: map[string]Observer{
			"view":     observeView,
			"debug":    observeDebug,
			"gostruct": observeGoStruct,
		},
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
		msg := d.runTeaCmd(cmd, trace)

		if msg != nil {
			rmsg := reflect.ValueOf(msg)
			if rmsg.Type().ConvertibleTo(cmdsType) {
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

func (d *driver) runTeaCmd(cmd tea.Cmd, trace bool) (res tea.Msg) {
	ctx, cancel := context.WithTimeout(d.ctx, d.cmdTimeout)
	defer cancel()

	msg := make(chan tea.Msg, 1)
	go func() {
		msg <- cmd()
	}()
	select {
	case <-ctx.Done():
		d.trace(trace, "timeout waiting for command")
	case res = <-msg:
	}
	return res
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

func (d *driver) Close(t TB) {
	d.cancel()
}

func (d *driver) RunOneTest(t TB, td *datadriven.TestData) string {
	// Save the input position.
	d.pos = td.Pos

	switch td.Cmd {
	case "set", "reset":
		return d.handleSet(t, td)
	case "run":
		return d.handleRun(t, td)
	default:
		t.Fatalf("%s: unrecognized test directive: %s", td.Pos, td.Cmd)
		panic("unreachable")
	}
}

func (d *driver) handleSet(t TB, td *datadriven.TestData) string {
	reset := td.Cmd == "reset"
	if len(td.CmdArgs) != 1 ||
		(!reset && len(td.CmdArgs[0].Vals) != 1) ||
		(reset && len(td.CmdArgs[0].Vals) != 0) {
		t.Fatalf("%s: invalid syntax", d.pos)
	}
	key := td.CmdArgs[0].Key
	val := ""
	if !reset {
		val = td.CmdArgs[0].Vals[0]
	}

	switch key {
	case "cmd_timeout":
		if reset {
			val = defaultCmdTimeout.String()
		}
		tm, err := time.ParseDuration(val)
		if err != nil {
			t.Fatalf("%s: invalid timeout value: %v", d.pos, err)
		}
		d.cmdTimeout = tm
		val = d.cmdTimeout.String()
	default:
		t.Fatalf("%s: unknown option %q", d.pos, key)
	}
	if reset {
		return "ok"
	}
	return fmt.Sprintf("%s: %s", key, val)
}

func (d *driver) handleRun(t TB, td *datadriven.TestData) string {
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

	default:
		obs, ok := d.observers[what]
		if !ok {
			t.Fatalf("%s: unsupported observer %q, did you call WithObserver()?", d.pos, what)
		}
		if err := obs(&buf, d.m); err != nil {
			t.Fatalf("%s: observing %q: %v", d.pos, what, err)
		}
	}
	return buf.String()
}

func observeView(buf io.Writer, m tea.Model) error {
	o := m.View()
	// Make newlines visible.
	o = strings.ReplaceAll(o, "\n", "␤\n")
	// Add a "no newline at end" marker if there was no newline at the end.
	if len(o) == 0 || o[len(o)-1] != '\n' {
		o += "🛇"
	}
	_, err := io.WriteString(buf, o)
	return err
}

func observeDebug(buf io.Writer, m tea.Model) error {
	type dbg interface{ Debug() string }
	md, ok := m.(dbg)
	if !ok {
		return errors.New("model does not support a Debug() string method")
	}
	_, err := io.WriteString(buf, md.Debug())
	return err
}

func observeGoStruct(buf io.Writer, m tea.Model) error {
	_, err := io.WriteString(buf, pretty.Sprint(m))
	return err
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
		d.addMsg(msg)

	case "key":
		d.assertArgc(t, args, 1)
		keyName := args[0]
		alt := false
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
		d.typeIn(args, alt)

	case "type":
		d.typeIn(args, false)

	case "enter":
		d.typeIn(args, false)
		d.addMsg(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))

	default:
		if d.upd != nil {
			t.Logf("%s: applying command %q via model updater", d.pos, cmd)
			supported, newModel, teaCmd, err := d.upd(d.m, cmd, args...)
			if err != nil {
				t.Fatalf("%s: updater error: %v", d.pos, err)
			}
			if !supported {
				t.Fatalf("%s: unknown command %q", d.pos, cmd)
			}
			d.m = newModel
			return teaCmd
		} else {
			t.Fatalf("%s: unknown command %q, and no Updater defined", d.pos, cmd)
		}
	}

	return nil
}

func (d *driver) typeIn(args []string, alt bool) {
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
