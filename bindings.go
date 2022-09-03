package catwalk

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// KeyMapUpdater defines an updater which supports the "keybind" and
// "keyhelp" commands to change a KeyMap struct. You can add this to
// a test using WithUpdater(). It is possible to add multiple keymap
// updaters to the same test.
//
// A KeyMap struct is any go struct containing exported fields of type
// key.Binding.  For example, using:
//
//    KeyMapUpdater("mymodel",
//                  func(m tea.Model, changeKeyMap func(interface{})) (tea.Model, err) {
//                     myModel := m.(mymodel)
//                     if err := changeKeyMap(&myModel.KeyMap); err != nil {
//                           return m, err
//                     }
//                     return myModel, nil
//                  })
//
// and mymodel.KeyMap containing a CursorUp binding,
// it becomes possible to use "keybind mymodel.CursorUp ctrl+c" to
// define a new keybinding during a test.
//
// If your model implements tea.Model by reference (i.e. its address
// does not change through Update calls), you can simplify
// the call as follows:
//
//      KeyMapUpdater("...", SimpleKeyMapApplier(&yourmodel.KeyMap)).
func KeyMapUpdater(prefix string, apply KeyMapApplier) Updater {
	return func(m tea.Model, inputCmd string, args ...string) (bool, tea.Model, tea.Cmd, error) {
		return handleKeyMapUpdate(prefix+".", apply, m, inputCmd, args...)
	}
}

// KeyMapApplier is the typpe of a function which applies the
// changeKeyMap callback on a KeyMap struct inside the model, then
// returns the resulting model.
//
// Example implementation:
//     func(m tea.Model, changeKeyMap func(interface{}) error) (tea.Model, err) {
//        myModel := m.(mymodel)
//        if err := changeKeyMap(&myModel.KeyMap); err != nil {
//              return m, err
//        }
//        return myModel, nil
//     }
type KeyMapApplier func(m tea.Model, changeKeyMap func(interface{}) error) (tea.Model, error)

// SimpleKeyMapApplier is a helper to simplify the definition of the
// function argument to KeyMapUpdater, in the case the model is
// implemented by reference -- i.e. the address of the KayMap does not
// change from one call to Update to the next.
func SimpleKeyMapApplier(keymap interface{}) KeyMapApplier {
	return func(m tea.Model, changeKeyMap func(interface{}) error) (tea.Model, error) {
		return m, changeKeyMap(keymap)
	}
}

func handleKeyMapUpdate(
	prefix string, apply KeyMapApplier, m tea.Model, inputcmd string, args ...string,
) (bool, tea.Model, tea.Cmd, error) {
	switch inputcmd {
	case "keybind":
		if len(args) < 2 {
			return false, m, nil, fmt.Errorf("syntax: keybind <bindingname> <newkey...>")
		}
		if !strings.HasPrefix(args[0], prefix) {
			// This keybind is meant for another updater. Not us.
			return false, m, nil, nil
		}
		bindingName := strings.TrimPrefix(args[0], prefix)
		newM, err := apply(m, func(km interface{}) error {
			return applyKeyRebind(km, bindingName, args[1:]...)
		})
		return true, newM, nil, err

	case "keyhelp":
		if len(args) < 3 {
			return false, m, nil, fmt.Errorf("syntax: keyhelp <bindingname> <helpkey> <helptext...>")
		}
		if !strings.HasPrefix(args[0], prefix) {
			// This keybind is meant for another updater. Not us.
			return false, m, nil, nil
		}
		bindingName := strings.TrimPrefix(args[0], prefix)
		newM, err := apply(m, func(km interface{}) error {
			return applyKeyNewHelp(km, bindingName, args[1], strings.Join(args[2:], " "))
		})
		return true, newM, nil, err

	default:
		// Command not supported.
		return false, m, nil, nil
	}
}

func applyKeyRebind(km interface{}, bindingName string, newKeys ...string) error {
	kb, err := getBinding(km, bindingName)
	if err != nil {
		return err
	}
	if len(newKeys) == 1 {
		switch newKeys[0] {
		case "enable":
			kb.SetEnabled(true)
			return nil
		case "disable":
			kb.SetEnabled(false)
			return nil
		case "unbind":
			kb.Unbind()
			return nil
		}
	}
	kb.SetKeys(newKeys...)
	return err
}

func applyKeyNewHelp(km interface{}, bindingName, helpKey, helpText string) error {
	kb, err := getBinding(km, bindingName)
	if err != nil {
		return err
	}
	kb.SetHelp(helpKey, helpText)
	return nil
}

func getBinding(km interface{}, bindingName string) (*key.Binding, error) {
	v := reflect.ValueOf(km)
	if v.Type().Kind() != reflect.Ptr {
		return nil, fmt.Errorf("keymap type %T is not a pointer to struct", km)
	}
	v = v.Elem()
	if v.Type().Kind() != reflect.Struct {
		return nil, fmt.Errorf("keymap type %T is not a pointer to struct", km)
	}
	var zv reflect.Value
	fv := v.FieldByName(bindingName)
	if fv == zv {
		return nil, fmt.Errorf("keymap struct %T does not contain a field named %q", km, bindingName)
	}
	if fv.Type() != keyBindingType {
		return nil, fmt.Errorf("field %q of struct %T does not have type key.Binding", bindingName, km)
	}
	return fv.Addr().Interface().(*key.Binding), nil
}

var keyBindingType = reflect.TypeOf(key.Binding{})
