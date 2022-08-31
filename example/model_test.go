package example

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/cockroachdb/datadriven"
	"github.com/knz/catwalk"
	"github.com/muesli/termenv"
)

func TestModel(t *testing.T) {
	// Initialize the model to test.
	m := New(40, 3)

	lipgloss.SetColorProfile(termenv.ANSI)
	m.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	// Use catwalk and datadriven to run all the tests in directory
	// "testdata".
	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		catwalk.RunModel(t, path, &m, catwalk.WithWindowSize(40, 3))
	})
}
