package example

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Model is a viewport adapted to follow the tea.Model interface.
type Model struct {
	viewport.Model
}

var _ tea.Model = (*Model)(nil)

// New initializes a new model.
func New(width, height int) Model {
	return Model{
		Model: viewport.New(width, height),
	}
}

const loremIpsum = `
lorem ipsum dolor sit amet, consectetur adipiscing
elit, sed do eiusmod tempor incididunt ut labore et dolore magna
aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco
laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor
in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla
pariatur. Excepteur sint occaecat cupidatat non proident, sunt in
culpa qui officia deserunt mollit anim id est laborum.

Sed ut perspiciatis unde omnis iste natus error sit voluptatem
accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae
ab illo inventore veritatis et quasi architecto beatae vitae dicta
sunt explicabo.`

func (m *Model) Init() tea.Cmd {
	s := strings.TrimSpace(loremIpsum)
	m.SetContent(s)
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newView, cmd := m.Model.Update(msg)
	m.Model = newView
	return m, cmd
}
