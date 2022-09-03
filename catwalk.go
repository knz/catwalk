package catwalk

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/datadriven"
)

// RunModel runs the tests contained in the file pointed to by 'path'
// on the model m, using a fresh driver initialize via NewDriver and
// the specified options.
//
// To apply RunModel on all the test files in a directory,
// use datadriven.Walk.
func RunModel(t *testing.T, path string, m tea.Model, opts ...Option) {
	t.Helper()
	d := NewDriver(m, opts...)
	defer d.Close(t)

	datadriven.RunTest(t, path, func(t *testing.T, td *datadriven.TestData) string {
		t.Helper()
		return d.RunOneTest(t, td)
	})
}

// RunModelFromString is a version of RunModel which takes the input
// test directives from a string directly.
func RunModelFromString(t *testing.T, input string, m tea.Model, opts ...Option) {
	t.Helper()
	d := NewDriver(m, opts...)
	defer d.Close(t)

	datadriven.RunTestFromString(t, input, func(t *testing.T, td *datadriven.TestData) string {
		t.Helper()
		return d.RunOneTest(t, td)
	})
}
