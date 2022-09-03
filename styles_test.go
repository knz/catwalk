package catwalk

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipglossc "github.com/knz/lipgloss-convert"
)

type mystyles struct {
	NotStyle int
	MyStyle  lipgloss.Style
	EmbeddedStyle
}

type EmbeddedStyle struct {
	OtherStyle lipgloss.Style
}

func TestGetStyle(t *testing.T) {
	var b mystyles
	var x int

	td := []struct {
		in     interface{}
		name   string
		out    *lipgloss.Style
		expErr string
	}{
		{b, "hello", nil, `type catwalk.mystyles is not a pointer to struct`},
		{&x, "hello", nil, `type *int is not a pointer to struct`},
		{&b, "hello", nil, `struct *catwalk.mystyles does not contain a field named "hello"`},
		{&b, "hello", nil, `struct *catwalk.mystyles does not contain a field named "hello"`},
		{&b, "NotStyle", nil, `field "NotStyle" of struct *catwalk.mystyles does not have type lipgloss.Style`},
		{&b, "MyStyle", &b.MyStyle, ``},
		{&b, "OtherStyle", &b.OtherStyle, ``},
	}

	for i, tc := range td {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			out, err := getStyle(tc.in, tc.name)
			if err != nil {
				if err.Error() != tc.expErr {
					t.Fatalf("expected error %q, got: %v", tc.expErr, err)
				}
				return
			}
			if err == nil && tc.expErr != "" {
				t.Fatalf("expected error %q, got no error", tc.expErr)
			}
			if out != tc.out {
				t.Fatalf("not valid style returned")
			}
		})
	}
}

func TestChangeStyle(t *testing.T) {
	var b mystyles

	err := applyStyleUpdate(&b, "hello", "foreground:11")
	if err == nil || err.Error() != `struct *catwalk.mystyles does not contain a field named "hello"` {
		t.Fatalf("restyle did not fail: %v", err)
	}
	err = applyStyleUpdate(&b, "MyStyle", "unsupported:11")
	if err == nil || err.Error() != `in "unsupported:11": property not supported: "unsupported"` {
		t.Fatalf("restyle did not fail: %v", err)
	}

	err = applyStyleUpdate(&b, "MyStyle", "foreground:11")
	if err != nil {
		t.Fatal(err)
	}
	s := lipglossc.Export(b.MyStyle)
	if s != "foreground: 11;" {
		t.Fatalf("style did not change properly: %s", s)
	}
}

// TestRestyle checks the restyle command.
func TestRestyle(t *testing.T) {
	t.Run("by-value", func(t *testing.T) {
		upd1 := StylesUpdater("view", func(m tea.Model, fn func(interface{}) error) (tea.Model, error) {
			h := m.(viewModel)
			err := fn(&h.viewport)
			return h, err
		})

		upd2 := StylesUpdater("model", func(m tea.Model, fn func(interface{}) error) (tea.Model, error) {
			h := m.(viewModel)
			err := fn(&h)
			return h, err
		})

		RunModel(t, "testdata/styles", newView(), WithUpdater(upd1), WithUpdater(upd2))
	})

	t.Run("by-reference", func(t *testing.T) {
		bv := newView()
		v := viewModelR(bv)
		upd1 := StylesUpdater("view", SimpleStylesApplier(&v.viewport))
		upd2 := StylesUpdater("model", SimpleStylesApplier(&v))

		RunModel(t, "testdata/styles", &v, WithUpdater(upd1), WithUpdater(upd2))
	})
}

type viewModel struct {
	val        int
	viewport   viewport.Model
	ValueStyle lipgloss.Style
}

func newView() viewModel {
	return viewModel{
		viewport: viewport.New(10, 3),
	}
}

func (h viewModel) Init() tea.Cmd { return nil }

func (h viewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	h.val++
	return h, nil
}

func (h viewModel) View() string {
	value := h.ValueStyle.Render(fmt.Sprintf("VALUE: %d", h.val))
	h.viewport.SetContent(value)
	return h.viewport.View()
}

type viewModelR viewModel

func (h *viewModelR) Init() tea.Cmd { return nil }
func (h *viewModelR) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newModel, newCmd := viewModel(*h).Update(msg)
	*h = viewModelR(newModel.(viewModel))
	return h, newCmd
}
func (h *viewModelR) View() string { return viewModel(*h).View() }
