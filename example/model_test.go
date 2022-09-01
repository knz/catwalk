package example

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/knz/catwalk"
	"github.com/muesli/termenv"
)

func TestModel(t *testing.T) {
	// Initialize the model to test.
	m := New(40, 3)

	lipgloss.SetColorProfile(termenv.Ascii)

	// Run all the tests in input file "testdata/viewport_tests"
	catwalk.RunModel(t, "testdata/viewport_tests", m)
}

func TestColors(t *testing.T) {
	// Initialize the model to test.
	m := New(40, 3)

	lipgloss.SetColorProfile(termenv.ANSI)
	m.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	catwalk.RunModel(t, "testdata/example", m, catwalk.WithWindowSize(40, 3))
}
