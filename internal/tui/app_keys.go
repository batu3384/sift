package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m appModel) matchesUp(msg tea.KeyMsg) bool        { return key.Matches(msg, m.keys.Up) }
func (m appModel) matchesDown(msg tea.KeyMsg) bool      { return key.Matches(msg, m.keys.Down) }
func (m appModel) matchesActivate(msg tea.KeyMsg) bool  { return key.Matches(msg, m.keys.Enter) }
func (m appModel) matchesHelp(msg tea.KeyMsg) bool      { return key.Matches(msg, m.keys.Help) }
func (m appModel) matchesTools(msg tea.KeyMsg) bool     { return key.Matches(msg, m.keys.Tools) }
func (m appModel) matchesBack(msg tea.KeyMsg) bool      { return key.Matches(msg, m.keys.Back) }
func (m appModel) matchesQuit(msg tea.KeyMsg) bool      { return key.Matches(msg, m.keys.Quit) }
func (m appModel) matchesOpen(msg tea.KeyMsg) bool      { return key.Matches(msg, m.keys.Open) }
func (m appModel) matchesStage(msg tea.KeyMsg) bool     { return key.Matches(msg, m.keys.Stage) }
func (m appModel) matchesUnstage(msg tea.KeyMsg) bool   { return key.Matches(msg, m.keys.Unstage) }
func (m appModel) matchesReveal(msg tea.KeyMsg) bool    { return key.Matches(msg, m.keys.Reveal) }
func (m appModel) matchesReview(msg tea.KeyMsg) bool    { return key.Matches(msg, m.keys.Review) }
func (m appModel) matchesRetry(msg tea.KeyMsg) bool     { return key.Matches(msg, m.keys.Retry) }
func (m appModel) matchesExecute(msg tea.KeyMsg) bool   { return key.Matches(msg, m.keys.Execute) }
func (m appModel) matchesCancel(msg tea.KeyMsg) bool    { return key.Matches(msg, m.keys.Cancel) }
func (m appModel) matchesStop(msg tea.KeyMsg) bool      { return key.Matches(msg, m.keys.Stop) }
func (m appModel) matchesRefresh(msg tea.KeyMsg) bool   { return key.Matches(msg, m.keys.Refresh) }
func (m appModel) matchesAdd(msg tea.KeyMsg) bool       { return key.Matches(msg, m.keys.Add) }
func (m appModel) matchesDelete(msg tea.KeyMsg) bool    { return key.Matches(msg, m.keys.Delete) }
func (m appModel) matchesExplain(msg tea.KeyMsg) bool   { return key.Matches(msg, m.keys.Explain) }
func (m appModel) matchesFilter(msg tea.KeyMsg) bool    { return key.Matches(msg, m.keys.Filter) }
func (m appModel) matchesSearch(msg tea.KeyMsg) bool    { return key.Matches(msg, m.keys.Search) }
func (m appModel) matchesSort(msg tea.KeyMsg) bool      { return key.Matches(msg, m.keys.Sort) }
func (m appModel) matchesFocus(msg tea.KeyMsg) bool     { return key.Matches(msg, m.keys.Focus) }
func (m appModel) matchesModule(msg tea.KeyMsg) bool    { return key.Matches(msg, m.keys.Module) }
func (m appModel) matchesCompanion(msg tea.KeyMsg) bool { return key.Matches(msg, m.keys.Companion) }
