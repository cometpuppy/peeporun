package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cometpuppy/peeporun/internal/config"
)

func (a *App) updatePresetSelect(key string) (tea.Model, tea.Cmd) {
	kb := a.kb
	n := len(a.presets)

	switch {
	case config.Matches(key, kb.Up):
		if n > 0 {
			a.selCursor--
			if a.selCursor < 0 {
				a.selCursor = n - 1
			}
		}
	case config.Matches(key, kb.Down):
		if n > 0 {
			a.selCursor = (a.selCursor + 1) % n
		}
	case config.Matches(key, kb.Confirm):
		if n > 0 {
			a.presetIdx = a.selCursor
			a.presetLoaded = true
			a.syncSaveShape()
			a.cursor = a.firstUnbeatenOrZero()
			a.screen = screenTracker
			a.persistState()
		}
	case config.Matches(key, kb.Edit):
		if n > 0 {
			a.editPresetIdx = a.selCursor
			a.editCursor = -2
			a.screen = screenPresetEdit
		}
	case config.Matches(key, kb.New):
		np := config.Preset{
			ID:       uniquePresetID(a.presets, "new-preset"),
			Game:     "",
			Category: "",
			Splits:   []string{},
		}
		a.presets = append(a.presets, np)
		a.autoNamedIDs[np.ID] = true
		a.editPresetIdx = len(a.presets) - 1
		a.editCursor = -2
		a.screen = screenPresetEdit
		a.persistPresets()
	case config.Matches(key, kb.Delete):
		if n > 0 {
			a.confirm = confirmDeletePreset
		}
	case config.Matches(key, kb.ThemePicker):
		a.screen = screenThemePicker
		a.themeCursor = 0
		a.themeEditMode = false
		a.input.SetValue("")
	case config.Matches(key, kb.Cancel):
		if a.presetLoaded {
			a.screen = screenTracker
		}
	}
	return a, nil
}

func (a *App) doDeletePreset() {
	if len(a.presets) == 0 {
		return
	}
	idx := a.selCursor
	id := a.presets[idx].ID
	a.presets = append(a.presets[:idx], a.presets[idx+1:]...)
	delete(a.save.Presets, id)

	if a.selCursor >= len(a.presets) {
		a.selCursor = len(a.presets) - 1
	}
	if a.selCursor < 0 {
		a.selCursor = 0
	}
	if a.presetIdx >= len(a.presets) {
		a.presetIdx = a.selCursor
	}
	a.persistPresets()
}

func uniquePresetID(presets []config.Preset, base string) string {
	id := base
	n := 1
	exists := func(id string) bool {
		for _, p := range presets {
			if p.ID == id {
				return true
			}
		}
		return false
	}
	for exists(id) {
		n++
		id = base + "-" + fmtInt(n)
	}
	return id
}

func (a *App) viewPresetSelect() string {
	var b strings.Builder
	b.WriteString(StyleGameTitle.Render("Presets"))
	b.WriteString("\n\n")

	if len(a.presets) == 0 {
		b.WriteString(StyleHelp.Render("No presets yet. Press 'n' to create one."))
	}

	for i, p := range a.presets {
		marker := "  "
		if i == a.selCursor {
			marker = "> "
		}
		game := p.Game
		if game == "" {
			game = "(untitled)"
		}
		row := marker + game
		if p.Category != "" {
			row += " \u2014 " + p.Category
		}
		if i == a.selCursor {
			b.WriteString(StyleActiveRow.Render(row))
		} else {
			b.WriteString(StyleNormalRow.Render(row))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	kb := a.kb
	parts := []string{
		pair(kb.Up, kb.Down) + " move",
		label(kb.Confirm, "select"),
		label(kb.Edit, "edit"),
		label(kb.New, "new"),
		label(kb.Delete, "delete"),
		label(kb.ThemePicker, "theme"),
	}
	if a.presetLoaded {
		parts = append(parts, label(kb.Cancel, "back"))
	}
	b.WriteString(StyleHelp.Render(joinHelp(parts...)))
	if a.status != "" {
		b.WriteString("\n")
		b.WriteString(StyleStatus.Render(a.status))
	}
	return StyleBox.Render(b.String())
}
