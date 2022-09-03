package catwalk

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type mybindings struct {
	NotBinding int
	MyBinding  key.Binding
	Embedded
}

type Embedded struct {
	OtherBinding key.Binding
}

func TestGetBinding(t *testing.T) {
	var b mybindings
	var x int

	td := []struct {
		in     interface{}
		name   string
		out    *key.Binding
		expErr string
	}{
		{b, "hello", nil, `keymap type catwalk.mybindings is not a pointer to struct`},
		{&x, "hello", nil, `keymap type *int is not a pointer to struct`},
		{&b, "hello", nil, `keymap struct *catwalk.mybindings does not contain a field named "hello"`},
		{&b, "hello", nil, `keymap struct *catwalk.mybindings does not contain a field named "hello"`},
		{&b, "NotBinding", nil, `field "NotBinding" of struct *catwalk.mybindings does not have type key.Binding`},
		{&b, "MyBinding", &b.MyBinding, ``},
		{&b, "OtherBinding", &b.OtherBinding, ``},
	}

	for i, tc := range td {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			out, err := getBinding(tc.in, tc.name)
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
				t.Fatalf("not valid binding returned")
			}
		})
	}
}

func TestChangeKeyHelp(t *testing.T) {
	var b mybindings

	err := applyKeyNewHelp(&b, "hello", "C-c", "break")
	if err == nil || err.Error() != `keymap struct *catwalk.mybindings does not contain a field named "hello"` {
		t.Fatalf("rebind did not fail: %v", err)
	}
	err = applyKeyNewHelp(&b, "MyBinding", "C-c", "break")
	if err != nil {
		t.Fatal(err)
	}
	h := b.MyBinding.Help()
	if h.Key != "C-c" || h.Desc != "break" {
		t.Fatalf("help did not change: %+v", h)
	}
}

func TestChangeKeyBinding(t *testing.T) {
	var b mybindings

	err := applyKeyRebind(&b, "hello", "ctrl+c")
	if err == nil || err.Error() != `keymap struct *catwalk.mybindings does not contain a field named "hello"` {
		t.Fatalf("rebind did not fail: %v", err)
	}

	err = applyKeyRebind(&b, "MyBinding", "ctrl+c")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(b.MyBinding.Keys(), []string{"ctrl+c"}) {
		t.Fatalf("binding not set properly: %v", b.MyBinding.Keys())
	}
	err = applyKeyRebind(&b, "MyBinding", "enable")
	if err != nil {
		t.Fatal(err)
	}
	if !b.MyBinding.Enabled() {
		t.Fatalf("binding not enabled")
	}
	err = applyKeyRebind(&b, "MyBinding", "disable")
	if err != nil {
		t.Fatal(err)
	}
	if b.MyBinding.Enabled() {
		t.Fatalf("binding not disabled")
	}
	err = applyKeyRebind(&b, "MyBinding", "unbind")
	if err != nil {
		t.Fatal(err)
	}
	if len(b.MyBinding.Keys()) > 0 {
		t.Fatalf("binding still bound: %v", b.MyBinding)
	}
}

// TestRebind checks the key rebind commands.
func TestRebind(t *testing.T) {
	t.Run("by-value", func(t *testing.T) {
		upd1 := KeyMapUpdater("hello", func(m tea.Model, fn func(interface{}) error) (tea.Model, error) {
			h := m.(helpModel)
			err := fn(&h.KeyMap)
			return h, err
		})

		upd2 := KeyMapUpdater("world", func(m tea.Model, fn func(interface{}) error) (tea.Model, error) {
			h := m.(helpModel)
			err := fn(&h.OtherKeyMap)
			return h, err
		})

		RunModel(t, "testdata/bindings", helpModel{}, WithUpdater(upd1), WithUpdater(upd2))
	})

	t.Run("by-reference", func(t *testing.T) {
		hm := &helpModelR{}
		upd1 := KeyMapUpdater("hello", SimpleKeyMapApplier(&hm.KeyMap))
		upd2 := KeyMapUpdater("world", SimpleKeyMapApplier(&hm.OtherKeyMap))

		RunModel(t, "testdata/bindings", hm, WithUpdater(upd1), WithUpdater(upd2))
	})
}

type helpModel struct {
	val    int
	help   help.Model
	KeyMap struct {
		MyKey key.Binding
	}
	OtherKeyMap struct {
		Other key.Binding
	}
}

func (h helpModel) Init() tea.Cmd { return nil }

func (h helpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	h.val++
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(kmsg, h.KeyMap.MyKey):
			return h, tea.Println("MYKEY RECOGNIZED")
		case key.Matches(kmsg, h.OtherKeyMap.Other):
			return h, tea.Println("OTHERKEY RECOGNIZED")
		default:
			return h, tea.Println("UNKOWN KEY")
		}
	}
	return h, nil
}

func (h helpModel) View() string {
	return fmt.Sprintf("VALUE: %d\n%s", h.val, h.help.View(h))
}

func (h helpModel) ShortHelp() []key.Binding {
	return []key.Binding{h.KeyMap.MyKey, h.OtherKeyMap.Other}
}

func (h helpModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{{h.KeyMap.MyKey}, {h.OtherKeyMap.Other}}
}

type helpModelR helpModel

func (h *helpModelR) Init() tea.Cmd { return nil }
func (h *helpModelR) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newModel, newCmd := helpModel(*h).Update(msg)
	*h = helpModelR(newModel.(helpModel))
	return h, newCmd
}
func (h *helpModelR) View() string { return helpModel(*h).View() }
