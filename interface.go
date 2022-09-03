package catwalk

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/datadriven"
)

// Updater is an optional function added to RunModel(), which can
// apply state change commands as input to a test.
//
// It should return false in the first return value to indicate that
// the command is not supported.
//
// It can return an error e.g. to indicate that the command is
// supported but its arguments use invalid syntax, or that the model
// is in an invalid state.
type Updater func(m tea.Model, testCmd string, args ...string) (supported bool, newModel tea.Model, teaCmd tea.Cmd, err error)

// Observer is an optional function added to RunModel(), which can
// extract information from the model to serve as expected output in
// tests.
type Observer func(out io.Writer, m tea.Model) error

// Option is the type of an option which can be specified
// with RunModel or NewDriver.
type Option func(*driver)

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
