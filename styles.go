package catwalk

import (
	"fmt"
	"reflect"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipglossc "github.com/knz/lipgloss-convert"
)

// StylesUpdater defines an updater which supports the "restyle"
// command to change a struct containing lipgloss.Style fields. You
// can add this to a test using WithUpdater(). It is possible to add
// multiple styles updaters to the same test.
//
// For example, using:
//
//    StylesUpdater("mymodel",
//                  func(m tea.Model, changeStyles func(interface{})) (tea.Model, err) {
//                     myModel := m.(mymodel)
//                     if err := changeStyles(&myModel); err != nil {
//                           return m, err
//                     }
//                     return myModel, nil
//                  })
//
// and mymodel containing a CursorStyle field,
// it becomes possible to use "restyle mymodel.CursorStyle foreground: 11" to
// define a new style during a test.
//
// If your model implements tea.Model by reference (i.e. its address
// does not change through Update calls), you can simplify
// the call as follows:
//
//      StylesUpdater("...", SimpleStylesApplier(&yourmodel)).
func StylesUpdater(prefix string, apply StylesApplier) Updater {
	return func(m tea.Model, inputCmd string, args ...string) (bool, tea.Model, tea.Cmd, error) {
		return handleStyleUpdate(prefix+".", apply, m, inputCmd, args...)
	}
}

// StylesApplier is the type of a function which applies the
// changeStyles callback on a struct inside the model, then
// returns the resulting model.
//
// Example implementation:
//     func(m tea.Model, changeStyles func(interface{}) error) (tea.Model, err) {
//        myModel := m.(mymodel)
//        if err := changeKeyMap(&myModel); err != nil {
//              return m, err
//        }
//        return myModel, nil
//     }
type StylesApplier func(m tea.Model, changeStyles func(interface{}) error) (tea.Model, error)

// SimpleSyylesApplier is a helper to simplify the definition of the
// function argument to StylesUpdater, in the case the model is
// implemented by reference -- i.e. the address of the styles does not
// change from one call to Update to the next.
func SimpleStylesApplier(styledStruct interface{}) StylesApplier {
	return func(m tea.Model, changeStyles func(interface{}) error) (tea.Model, error) {
		return m, changeStyles(styledStruct)
	}
}

func handleStyleUpdate(
	prefix string, apply StylesApplier, m tea.Model, inputcmd string, args ...string,
) (bool, tea.Model, tea.Cmd, error) {
	switch inputcmd {
	case "restyle":
		if len(args) < 2 {
			return false, m, nil, fmt.Errorf("syntax: restyle <stylename> <style...>")
		}
		if !strings.HasPrefix(args[0], prefix) {
			// This keybind is meant for another updater. Not us.
			return false, m, nil, nil
		}
		styleName := strings.TrimPrefix(args[0], prefix)
		newM, err := apply(m, func(km interface{}) error {
			return applyStyleUpdate(km, styleName, strings.Join(args[1:], " "))
		})
		return true, newM, nil, err

	default:
		// Command not supported.
		return false, m, nil, nil
	}
}

func applyStyleUpdate(km interface{}, styleName, newStyle string) error {
	s, err := getStyle(km, styleName)
	if err != nil {
		return err
	}
	sres, err := lipglossc.Import(*s, newStyle)
	if err != nil {
		return err
	}
	*s = sres
	return nil
}

func getStyle(km interface{}, styleName string) (*lipgloss.Style, error) {
	v := reflect.ValueOf(km)
	if v.Type().Kind() != reflect.Ptr {
		return nil, fmt.Errorf("type %T is not a pointer to struct", km)
	}
	v = v.Elem()
	if v.Type().Kind() != reflect.Struct {
		return nil, fmt.Errorf("type %T is not a pointer to struct", km)
	}
	var zv reflect.Value
	fv := v.FieldByName(styleName)
	if fv == zv {
		return nil, fmt.Errorf("struct %T does not contain a field named %q", km, styleName)
	}
	if fv.Type() != keyStyleType {
		return nil, fmt.Errorf("field %q of struct %T does not have type lipgloss.Style", styleName, km)
	}
	return fv.Addr().Interface().(*lipgloss.Style), nil
}

var keyStyleType = reflect.TypeOf(lipgloss.NewStyle())
