package catwalk

import tea "github.com/charmbracelet/bubbletea"

// WithAutoInitDisabled tells the test driver to not automatically
// initialize the model (via the Init method) upon first use.
func WithAutoInitDisabled() Option {
	return func(d *driver) {
		d.disableAutoInit = true
	}
}

// WithWindowSize tells the test driver to issue a tea.WindowSizeMsg
// as the first event after initialization.
func WithWindowSize(width, height int) Option {
	return func(d *driver) {
		d.autoSize = true
		d.width = width
		d.height = height
	}
}

// WithObserver tells the test driver to support an additional
// observer with the given function.
//
// For example, after WithObserver("hello", myObserver)
// The function myObserver() will be called every time
// a test specifies `observe=hello` in the run directive.
func WithObserver(what string, obs Observer) Option {
	return func(d *driver) {
		d.observers[what] = obs
	}
}

// WithUpdater adds the specified model updater to the test.
// It is possible to use multiple WithUpdater options, which will
// chain them automatically (using ChainUpdaters).
func WithUpdater(upd Updater) Option {
	return func(d *driver) {
		d.upd = ChainUpdaters(d.upd, upd)
	}
}

// ChainUpdaters chains the specified updaters into a resulting updater
// that supports all the commands in the chain. Test input commands
// are passed to each updater in turn until the first updater
// that supports it.
//
// For example:
// - upd1 supports command "print"
// - upd2 supports command "get"
// - ChainUpdaters(upd1, upd2) will support both commands "print" and "get.
func ChainUpdaters(upds ...Updater) Updater {
	actual := make([]Updater, 0, len(upds))
	for _, u := range upds {
		if u != nil {
			actual = append(actual, u)
		}
	}
	if len(actual) == 1 {
		return actual[0]
	}
	return func(m tea.Model, inputCmd string, args ...string) (supported bool, newModel tea.Model, teaCmd tea.Cmd, err error) {
		for _, upd := range actual {
			supported, newModel, teaCmd, err = upd(m, inputCmd, args...)
			if supported || err != nil {
				return supported, newModel, teaCmd, err
			}
		}
		// None of the updaters supported the command.
		return false, nil, nil, nil
	}
}
