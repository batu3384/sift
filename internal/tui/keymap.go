package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

type appKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	Help      key.Binding
	Tools     key.Binding
	Back      key.Binding
	Quit      key.Binding
	Open      key.Binding
	Stage     key.Binding
	Unstage   key.Binding
	Reveal    key.Binding
	Review    key.Binding
	Retry     key.Binding
	Execute   key.Binding
	Cancel    key.Binding
	Stop      key.Binding
	Refresh   key.Binding
	Add       key.Binding
	Delete    key.Binding
	Explain   key.Binding
	Filter    key.Binding
	Search    key.Binding
	Sort      key.Binding
	Focus     key.Binding
	Module    key.Binding
	Companion key.Binding
}

type routeHelp struct {
	short    []key.Binding
	sections []helpSection
}

type helpSection struct {
	title    string
	bindings []key.Binding
}

func defaultKeyMap() appKeyMap {
	return appKeyMap{
		Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:     key.NewBinding(key.WithKeys("enter", "right", "l"), key.WithHelp("enter", "open")),
		Help:      key.NewBinding(key.WithKeys("?", "f1"), key.WithHelp("?", "help")),
		Tools:     key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "tools")),
		Back:      key.NewBinding(key.WithKeys("esc", "left", "h", "backspace"), key.WithHelp("esc", "back")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Open:      key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open")),
		Stage:     key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "stage")),
		Unstage:   key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "unstage")),
		Reveal:    key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "reveal")),
		Review:    key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "review")),
		Retry:     key.NewBinding(key.WithKeys("ctrl+r", "R"), key.WithHelp("ctrl+r", "retry")),
		Execute:   key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "execute")),
		Cancel:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "cancel")),
		Stop:      key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "stop")),
		Refresh:   key.NewBinding(key.WithKeys("f5", "r"), key.WithHelp("f5/r", "refresh")),
		Add:       key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		Delete:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "remove")),
		Explain:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "explain")),
		Filter:    key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "filter")),
		Search:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Sort:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
		Focus:     key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("tab", "focus")),
		Module:    key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "module")),
		Companion: key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "companion")),
	}
}

func newHelpModel() help.Model {
	h := help.New()
	h.ShowAll = false
	return h
}
