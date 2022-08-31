package example

import (
	"testing"

	"github.com/cockroachdb/datadriven"
	"github.com/knz/catwalk"
)

func TestModel(t *testing.T) {
	// Initialize the model to test.
	m := New(40, 10)

	// Use catwalk and datadriven to run all the tests in directory
	// "testdata".
	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		catwalk.RunModel(t, path, m)
	})
}
