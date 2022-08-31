# catwalk

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/knz/catwalk)
[![Go ReportCard](https://goreportcard.com/badge/knz/catwalk)](https://goreportcard.com/report/knz/catwalk)

**catwalk** is a unit test library for Bubbletea TUI models.

It enables implementers to verify the state of models as
they process `tea.Msg` objects through their `Update` method.

It is implemented on top of
[datadriven](https://github.com/cockroachdb/datadriven), an extension
to Go's simple "table-driven testing" idiom.

Datadriven tests use data files containing both the reference input
and output, instead of a data structure. Each data file can contain
multiple tests. When the implementation changes, the reference output
can be quickly updated by re-running the tests with the `-rewrite`
flag.


## Example use

Place in the go test file:

``` go
func TestModel(t *testing.T) {
    // Define the model to test.
    m := NewModel(40, 3)

    // Optional: force output color codes, to debug colorization.
    lipgloss.SetColorProfile(termenv.ANSI)

    // Use catwalk and datadriven to run all the tests in directory
    // "testdata".
    datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
        catwalk.RunModel(t, path, &m)
    })
}
```

Then, place in a subdirectory called `testdata`, a text file with
some arbitrary name (e.g. `example`), containing:

``` text
# Just a run directive without command is a no-op
# and, by default, tests the resulting view.
run
----

# A run directive that inputs the "q" keypress.
run
type q
----
```

Then, run your test with `go test . -args -rewrite` to generate the
reference output.

Then reload the test input file you've created above in your text
editor, and observe: the test framework has updated the expected output!

Then, run your test again with `go test .` (without `-rewrite`!) to verify
that your model still matches its expected output.

See the `example` subdirectory in this repository for a concrete
example applied to the `viewport` bubble.
