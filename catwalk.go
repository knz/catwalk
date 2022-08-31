package catwalk

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/datadriven"
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
}

// Driver is the externally-visible interface for a test driver.
type Driver interface {
	// Start initializes the model and prepares it for testing.
	Start(t *testing.T)
	// Close stops the model at the end.
	Close(t *testing.T)
	// RunOneTest runs one step of a test file.
	RunOneTest(t *testing.T, d *datadriven.TestData) string
}

// NewDriver creates a test driver for the given model.
func NewDriver(m tea.Model) Driver {
	return &driver{m: m}
}

func (d *driver) Start(t *testing.T) {
	// Something m.Init().
}

func (d *driver) Close(t *testing.T) {}

func (d *driver) RunOneTest(t *testing.T, td *datadriven.TestData) string {
	return ""
}
