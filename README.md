# catwalk

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/knz/catwalk)
[![Build Status](https://github.com/knz/catwalk/workflows/build/badge.svg)](https://github.com/knz/catwalk/actions)
[![Go ReportCard](https://goreportcard.com/badge/knz/catwalk)](https://goreportcard.com/report/knz/catwalk)
[![Coverage Status](https://coveralls.io/repos/github/knz/catwalk/badge.svg)](https://coveralls.io/github/knz/catwalk)

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

## Example

Let's test the `viewport` Bubble!

First, we define a top-level model around `viewport`:

```go
type Model struct {
    viewport.Model
}

var _ tea.Model = (*Model)(nil)

// New initializes a new model.
func New(width, height int) *Model {
    return &Model{
        Model: viewport.New(width, height),
    }
}

// Init adds some initial text inside the viewport.
func (m *Model) Init() tea.Cmd {
    cmd := m.Model.Init()
    m.SetContent(`first line
second line
third line
fourth line
fifth line
sixth line`)
    return cmd
}
```

Then, we define a Go test which runs the above model:

```go
func TestModel(t *testing.T) {
	// Initialize the model to test.
	m := New(40, 3)

    // Run all the tests in input file "testdata/viewport_tests"
	catwalk.RunModel(t, "testdata/viewport_tests", m)
}
```

Then, we populate some test directives inside `testdata/viewport_tests`:

``` go
run
----

# One line down
run
type j
----

# Two lines down
run
type jj
----

# One line up
run
key up
----
```

Then, we run the test: `go test .`.

What happens: the test fails!

```
--- FAIL: TestModel (0.00s)
    catwalk.go:64:
        testdata/viewport_tests:1:
        expected:

        found:
        -- view:
        first line␤
        second line␤
        third line🛇
```

This is because we haven't yet expressed
what is the **expected output** for each step.

Because it's tedious to do this manually, we can auto-generate
the expected output from the actual output, using the `rewrite` flag:

     go test . -args -rewrite

Observe what happened with the input file:

``` go
run
----
-- view:
first line␤
second line␤
third line🛇

# One line down
run
type j
----
-- view:
second line␤
third line␤
fourth line🛇

# Two lines down
run
type jj
----
-- view:
fourth line␤
fifth line␤
sixth line🛇

# One line up
run
key up
----
-- view:
third line␤
fourth line␤
fifth line🛇
```

Now each expected output reflects how the `viewport` reacts
to the key presses. Now also `go test .` succeeds.
