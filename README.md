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

**Why the name:** the framework forces the Tea models to "show
themselves" on the runway of the test input files.

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

## Structure of a test file

Test files contain zero or more tests, with the following structure:

``` go
<directive> <arguments>
<optional: input commands...>
----
<expected output>
```

For example:

``` go
run
----
-- view:
My bubble rendered here.

run
type q
----
-- view:
My bubble reacted to "q".
```

Catwalk supports the `run` directive, which applies state changes to the
model via its `Update` method.

Under `run`, the following input commands are supported:

- `type <text>`: produce a series of `tea.KeyMsg` with type
  `tea.KeyRunes`. Can contain spaces.

  For example: `type abc` produces 3 key presses for a, b, c.

- `key <keyname>`: produce one `tea.KeyMsg` for the given key.

  For example: `key ctrl+c`

- `resize <W> <H>`: produce a `tea.WindowSizeMsg` with the specified size.

You can also add support for your own input commands by passing an
`Updater` function to `catwalk.RunModel` with the `WithUpdater()`
option, and combine multiple updaters together using the
`ChainUpdater()` function.

Finally, directives can take arguments. For example:

```
run observe=(gostruct,view)
----
```

The `run` directive accepts the following arguments:

- `observe`: what to look at as expected output (`observe=xx` or `observe=(xx,yy)`).

  By default, `observe` is set to `view`: look at the model's `View()` method.
  Alternatively, you can use the following observers:

  - `gostruct`: show the contents of the model object as a go struct.
  - `debug`: call the model's `Debug() string` method, if defined.

- `trace`: detail the intermediate steps of the test.

  Used for debugging tests.

You can also add your own observers using the `WithObserver()` option.

## Your turn!

You can start using `catwalk` in your Bubbletea / Charm projects right
away!

If you have any questions or comments:

- for bug fixes, feature requests, etc., [file an issue]()
- for questions, suggestions, etc. you can come chat on the [Charm
  Slack](https://charm.sh/slack/).
