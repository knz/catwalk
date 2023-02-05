# catwalk

[![Latest Release](https://img.shields.io/github/release/knz/catwalk.svg)](https://github.com/knz/catwalk/releases)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/knz/catwalk)
[![Build Status](https://github.com/knz/catwalk/workflows/build/badge.svg)](https://github.com/knz/catwalk/actions)
[![Go ReportCard](https://goreportcard.com/badge/knz/catwalk)](https://goreportcard.com/report/knz/catwalk)
[![Coverage Status](https://coveralls.io/repos/github/knz/catwalk/badge.svg)](https://coveralls.io/github/knz/catwalk)

**catwalk** is a unit test library for [Bubbletea](https://github.com/charmbracelet/bubbletea) TUI models (a.k.a. “bubbles”).

It enables implementers to verify the state of models and their `View`
as they process `tea.Msg` objects through their `Update` method.

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

Let's test the `viewport` [Bubble](https://github.com/charmbracelet/bubbles)!

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

Catwalk supports the following directives:

- `run`: apply state changes to the  model via its `Update` method, then show the results.
- `set`/`reset`: change configuration variables.

Finally, directives can take arguments. For example:

```
run observe=(gostruct,view)
----
```

This is explained further in the next sections.

## The `run` directive

`run` defines one unit test. It applies some input commands to the
model then compares the resulting state of the model with a reference
expected output.

Under `run`, the following input commands are supported:

- `type <text>`: produce a series of `tea.KeyMsg` with type
  `tea.KeyRunes`. Can contain spaces.

  For example: `type abc` produces 3 key presses for a, b, c.

- `enter <text>`: like `type`, but also add a key press for the
  `enter` key at the end.

- `key <keyname>`: produce one `tea.KeyMsg` for the given key.

  For example: `key ctrl+c`

- `paste "<text>"`: paste the text as a single key event.
  The text can contain Go escape sequences.

  For example: `paste "hello\nworld"`

- `resize <W> <H>`: produce a `tea.WindowSizeMsg` with the specified size.

You can also add support for your own input commands by passing an
`Updater` function to `catwalk.RunModel` with the `WithUpdater()`
option, and combine multiple updaters together using the
`ChainUpdater()` function.

The `run` directive accepts the following arguments:

- `observe`: what to look at as expected output (`observe=xx` or `observe=(xx,yy)`).

  By default, `observe` is set to `view`: look at the model's `View()` method.
  Alternatively, you can use the following observers:

  - `gostruct`: show the contents of the model object as a go struct.
  - `debug`: call the model's `Debug() string` method, if defined.

  You can also add your own observers using the `WithObserver()` option.

- `trace`: detail the intermediate steps of the test.

  Used for debugging tests.

## The `set` and `reset` directives

These can be used to configure parameters in the test driver.

For example:

``` go
set cmd_timeout=100ms
----
cmd_timeout: 100ms

reset cmd_timeout
----
ok
```

The following parameters are currently recognized:

- `cmd_timeout`: how long to wait for a `tea.Cmd` to complete.
  This is set by default to 20ms, which is sufficient to
  ignore the commands of a blinking cursor.

## Advanced topic: testing style changes

Many [bubbles](https://github.com/charmbracelet/bubbles) have a
`Styles` struct with configurable styles (using [lipgloss](https://github.com/charmbracelet/lipgloss)).  It's useful to verify
that the bubbles react properly when the styles are reconfigured at
run-time.

For this, you can tell catwalk about your styles
this will activate the following special `run` input commands:

```
restyle <stylefield> <newstyle...>`
```


For example: `restyle mymodel.ValueStyle foreground: #f00` changes the
`ValueStyle` style to use the color red, as if `.ValueStyle.Foreground(lipgloss.Color("#f00"))` was called.

To activate, use the option `catwalk.WithUpdater(catwalk.StylesUpdater(...))`. For example:

``` go
func TestStyles(t *testing.T) {
  m := New(...)
  catwalk.RunModel(t, "testdata/styles", m, catwalk.WithUpdater(
    // The string "hello" is the prefix for identifying the styles container in tests.
    // Useful when there are multiple nested models.
    catwalk.StylesUpdater("hello",
      func(m tea.Model, fn func(interface{}) error) (tea.Model, error) {
        tm := m.(myModel)
        err := fn(&tm)
        return tm, err
    }),
  ))
}
```

After this, the input command `restyle hello.X ...` will automatically
affect the style `.X` in your model.

Alternatively, if your model implements `tea.Model` by reference
(i.e. the address of its styles does not change between `Update`
calls), you can simplify as follows:

``` go
func TestStyles(t *testing.T) {
  m := New(...)
  catwalk.RunModel(t, "testdata/bindings", &m, catwalk.WithUpdater(
    // The string "hello" is the prefix for identifying the styles container in tests.
    // Useful when there are multiple nested models.
    KeyMapUpdater("hello", catwalk.SimpleStylesApplier(&m))))
}
```

See the test `TestStyles` in `styles_test.go` and the input file
`testdata/styles` for an example.

## Advanced topic: testing key bindings

Many [bubbles](https://github.com/charmbracelet/bubbles) have a
`KeyMap` struct with configurable key bindings.  It's useful to verify
that the bubbles react properly when the keymaps are reconfigured at
run-time.

For this, you can tell catwalk about your `KeyMap` struct and
this will activate the following special `run` input commands:

- `keybind <keymapfield> <newbinding>`

  For example: `keybind mykeys.CursorUp up j` rebinds the `CursorUp`
  binding in the KeyMap `mykeys` as if
  `key.NewBinding(key.WithKeys("up", "j"))` was called.

- `keyhelp <keymapfield> <helpkey> <helptext>`

  For example: `keybind mykeys.CursorUp up move the cursor up` rebinds
  the `CursorUp` binding in the KeyMap `mykeys` as if
  `key.NewBinding(key.WithHelp("up", "move the cursor up"))` was
  called.

To declare a `KeyMap` in a test, use the option `catwalk.WithUpdater(catwalk.KeyMapUpdater(...))`. For example:

``` go
func TestBindings(t *testing.T) {
  m := New(...)
  catwalk.RunModel(t, "testdata/bindings", m, catwalk.WithUpdater(
    // The string "hello" is the prefix for identifying the keymap in tests.
	// Useful when the model contains multiple keymaps.
    catwalk.KeyMapUpdater("hello",
      func(m tea.Model, fn func(interface{}) error) (tea.Model, error) {
        tm := m.(YourModel)
        err := fn(&tm.KeyMap)
        return tm, err
    }),
  ))
}
```

After this, the input command `keybind hello.X ...` will automatically
affect the binding `.KeyMap.X` in your model.

Alternatively, if your model implements `tea.Model` by reference (i.e. the address of its KeyMap does not change between `Update` calls), you can simplify as follows:

``` go
func TestBindings(t *testing.T) {
  m := New(...)
  catwalk.RunModel(t, "testdata/bindings", &m, catwalk.WithUpdater(
    // The string "hello" is the prefix for identifying the keymap in tests.
	// Useful when the model contains multiple keymaps.
    KeyMapUpdater("hello", catwalk.SimpleKeyMapApplier(&m.KeyMap))))
}
```

See the test `TestRebind` in `bindings_test.go` and the input file
`testdata/bindings` for an example.

## Your turn!

You can start using `catwalk` in your Bubbletea / Charm projects right
away!

If you have any questions or comments:

- for bug fixes, feature requests, etc., [file an issue]()
- for questions, suggestions, etc. you can come chat on the [Charm
  Discord](https://charm.sh/chat/).
