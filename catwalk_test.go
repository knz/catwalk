package catwalk

import (
	"fmt"
	"strconv"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestModel checks basic features.
func TestModel(t *testing.T) {
	RunModel(t, "testdata/simple", emptyModel{})
}

type emptyModel struct{}

var _ tea.Model = emptyModel{}

func (emptyModel) Init() tea.Cmd {
	return tea.Println("MODEL INIT")
}
func (emptyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kmsg, ok := msg.(tea.KeyMsg); ok && kmsg.Type == tea.KeyRunes {
		switch string(kmsg.Runes) {
		case "q":
			return emptyModel{}, tea.Quit
		case "M":
			return emptyModel{}, tea.DisableMouse
		case "m":
			return emptyModel{}, tea.EnableMouseAllMotion
		case "c":
			return emptyModel{}, tea.EnableMouseCellMotion
		case "a":
			return emptyModel{}, tea.EnterAltScreen
		case "A":
			return emptyModel{}, tea.ExitAltScreen
		case "C":
			return emptyModel{}, tea.HideCursor
		case "x":
			return emptyModel{}, tea.ExecProcess(nil, nil)
		case "e":
			return emptyModel{}, func() tea.Msg { return nil }
		}
	}
	return emptyModel{}, tea.Println("MODEL UPDATE")
}
func (emptyModel) View() string { return "MODEL VIEW" }

// TestDisableAutoInit checks the WithAutoInitDisabled configuration option.
func TestDisableAutoInit(t *testing.T) {
	RunModel(t, "testdata/disable_auto_start", emptyModel{}, WithAutoInitDisabled())
}

// TestInitWindowSize checks that a WindowSizeMsg is sent at the first interaction.
func TestInitWindowSize(t *testing.T) {
	RunModel(t, "testdata/window_size", emptyModel{}, WithWindowSize(80, 25))
}

// TestModelThreading checks that catwalk preserves the model returned
// by the Update function.
func TestModelThreading(t *testing.T) {
	RunModel(t, "testdata/model_threading", intModel(0), WithUpdater(updater{}))
}

type intModel int

var _ tea.Model = intModel(0)

func (intModel) Init() tea.Cmd {
	return nil
}
func (m intModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	m++
	return m, nil
}
func (m intModel) View() string { return "VALUE: " + strconv.Itoa(int(m)) }

type updater struct{}

var _ ModelUpdater = updater{}

func (updater) TestUpdate(t TB, m tea.Model, cmd string, args ...string) (tea.Model, tea.Cmd) {
	im := m.(intModel)
	if cmd == "double" {
		im = im * 2
	}
	return im, tea.Printf("TEST UPDATE CALLED WITH %s %v", cmd, args)
}

// TestCmdExpansion checks that tea.Batch and tea.Sequence are processed
// properly.
func TestCmdExpansion(t *testing.T) {
	RunModel(t, "testdata/expansion", cmdModel{}, WithUpdater(cmdUpdater{}))
}

type cmdModel struct{}

var _ tea.Model = cmdModel{}

func (cmdModel) Init() tea.Cmd {
	return tea.Batch(
		tea.Println("init1"), func() tea.Msg { return nil },
		tea.Sequence(tea.Println("init2"), tea.Println("init3")))
}
func (cmdModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return cmdModel{}, tea.Batch(
		tea.Println("upd1"), func() tea.Msg { return nil },
		tea.Sequence(tea.Println("upd2"), tea.Println("upd3")))
}
func (cmdModel) View() string { return "" }

type cmdUpdater struct{}

var _ ModelUpdater = cmdUpdater{}

func (cmdUpdater) TestUpdate(t TB, m tea.Model, cmd string, args ...string) (tea.Model, tea.Cmd) {
	return m, tea.Batch(
		tea.Println("tupd1"),
		tea.Sequence(tea.Println("tupd2"), tea.Println("tupd3")))
}

// TestObserve tests the various accepted values for the "observe"
// directive option.
func TestObserve(t *testing.T) {
	RunModel(t, "testdata/observe", &structModel{})
}

type structModel struct{ x int }

var _ tea.Model = (*structModel)(nil)

func (s *structModel) Init() tea.Cmd {
	s.x = 4242
	return nil
}
func (s *structModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	s.x++
	return s, nil
}
func (s *structModel) View() string { return fmt.Sprintf("VALUE: %q", s.x) }

func (s *structModel) Debug() string { return "DEBUG SAYS HI" }
