package catwalk

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestDisableAutoInit checks the WithAutoInitDisabled configuration option.
func TestDisableAutoInit(t *testing.T) {
	RunModel(t, "testdata/disable_auto_start", emptyModel{}, WithAutoInitDisabled())
}

// TestInitWindowSize checks that a WindowSizeMsg is sent at the first interaction.
func TestInitWindowSize(t *testing.T) {
	RunModel(t, "testdata/window_size", emptyModel{}, WithWindowSize(80, 25))
}

func TestChainUpdaters(t *testing.T) {
	upd1 := func(_ tea.Model, cmd string, _ ...string) (bool, tea.Model, tea.Cmd, error) {
		if cmd == "hello" {
			return true, nil, nil, nil
		}
		return false, nil, nil, nil
	}
	upd2 := func(_ tea.Model, cmd string, _ ...string) (bool, tea.Model, tea.Cmd, error) {
		if cmd == "world" {
			return true, nil, nil, nil
		}
		return false, nil, nil, nil
	}

	upd := ChainUpdaters(upd1, upd2)

	if s, _, _, _ := ChainUpdaters(upd1)(nil, "hello"); !s {
		t.Errorf("updater doesn't propagate single argument")
	}
	if s, _, _, _ := upd(nil, "hello"); !s {
		t.Errorf("first updater did not register")
	}
	if s, _, _, _ := upd(nil, "world"); !s {
		t.Errorf("2nd updater did not register")
	}
	if s, _, _, _ := upd(nil, "unknown"); s {
		t.Errorf("surprising updater result")
	}
	if s, _, _, _ := ChainUpdaters()(nil, "unknown"); s {
		t.Errorf("surprising updater result")
	}
}

func TestChainComplexUpdaters(t *testing.T) {
	hm := &helpModelR{}
	upd1 := KeyMapUpdater("hello", SimpleKeyMapApplier(&hm.KeyMap))
	upd2 := StylesUpdater("help", SimpleStylesApplier(&hm.help.Styles))
	upd3 := func(m tea.Model, cmd string, args ...string) (bool, tea.Model, tea.Cmd, error) {
		m.(*helpModelR).val = 123
		return true, m, nil, nil
	}

	const test = `
run
keybind hello.MyKey ctrl+c
keyhelp hello.MyKey C-c break
restyle help.Ellipsis foreground: #f00
othercmd
----
-- view:
VALUE: 123␤
C-c break🛇`

	RunModelFromString(t, test, hm, WithUpdater(upd1), WithUpdater(upd2), WithUpdater(upd3))
}
