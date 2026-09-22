package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cometpuppy/peeporun/internal/config"
)

func (a *App) updateTracker(key string) (tea.Model, tea.Cmd) {
	kb := a.kb
	p := a.currentPreset()

	switch {
	case config.Matches(key, kb.Up):
		if len(p.Splits) > 0 {
			a.cursor--
			if a.cursor < 0 {
				a.cursor = len(p.Splits) - 1
			}
		}
	case config.Matches(key, kb.Down):
		if len(p.Splits) > 0 {
			a.cursor = (a.cursor + 1) % len(p.Splits)
		}
	case config.Matches(key, kb.Hit):
		a.addHit(1)
	case config.Matches(key, kb.Undo):
		a.addHit(-1)
	case config.Matches(key, kb.Split):
		a.beatCurrentAndAdvance()
	case config.Matches(key, kb.Unsplit):
		a.unsplit()
	case config.Matches(key, kb.SavePB):
		a.confirm = confirmSavePB
	case config.Matches(key, kb.DeletePB):
		a.confirm = confirmDeletePB
	case config.Matches(key, kb.Reset):
		a.confirm = confirmReset
	case config.Matches(key, kb.Presets):
		a.screen = screenPresetSelect
		a.selCursor = a.presetIdx
	}
	return a, nil
}

func (a *App) addHit(delta int) {
	p := a.currentPreset()
	if len(p.Splits) == 0 {
		return
	}
	ps := a.save.Presets[p.ID]
	h := ps.Current[a.cursor].Hits + delta
	if h < 0 {
		h = 0
	}
	ps.Current[a.cursor].Hits = h
	a.save.Presets[p.ID] = ps
	a.persistState()
}

func (a *App) beatCurrentAndAdvance() {
	p := a.currentPreset()
	if len(p.Splits) == 0 {
		return
	}
	ps := a.save.Presets[p.ID]
	ps.Current[a.cursor].Beaten = true
	a.save.Presets[p.ID] = ps

	// last split just beaten -> offer PB save
	allBeaten := true
	for _, s := range ps.Current {
		if !s.Beaten {
			allBeaten = false
			break
		}
	}

	if a.cursor < len(p.Splits)-1 {
		a.cursor++
	}
	a.persistState()

	if allBeaten {
		total := 0
		for _, s := range ps.Current {
			total += s.Hits
		}
		if !ps.HasPB || total < ps.PBTotal {
			a.confirm = confirmSavePB
		}
	}
}

func (a *App) unsplit() {
	p := a.currentPreset()
	if len(p.Splits) == 0 {
		return
	}
	ps := a.save.Presets[p.ID]
	if ps.Current[a.cursor].Beaten {
		ps.Current[a.cursor].Beaten = false
	} else if a.cursor > 0 {
		a.cursor--
		ps.Current[a.cursor].Beaten = false
	}
	a.save.Presets[p.ID] = ps
	a.persistState()
}

func (a *App) resetRun() {
	p := a.currentPreset()
	if len(p.Splits) == 0 {
		return
	}
	ps := a.save.Presets[p.ID]
	ps.Current = make([]config.SplitState, len(p.Splits))
	a.save.Presets[p.ID] = ps
	a.cursor = 0
	a.persistState()
}

func (a *App) savePB() {
	p := a.currentPreset()
	ps := a.save.Presets[p.ID]
	pb := make([]config.PBEntry, len(p.Splits))
	total := 0
	for i, s := range ps.Current {
		pb[i] = config.PBEntry{Hits: s.Hits}
		total += s.Hits
	}
	ps.PB = pb
	ps.PBTotal = total
	ps.HasPB = true
	a.save.Presets[p.ID] = ps
	a.persistState()
}

func (a *App) deletePB() {
	p := a.currentPreset()
	ps := a.save.Presets[p.ID]
	ps.PB = make([]config.PBEntry, len(p.Splits))
	ps.PBTotal = 0
	ps.HasPB = false
	a.save.Presets[p.ID] = ps
	a.persistState()
}

func (a *App) viewTracker() string {
	p := a.currentPreset()
	ps := a.save.Presets[p.ID]

	var b strings.Builder
	gameTitle := p.Game
	if gameTitle == "" {
		gameTitle = "(untitled)"
	}
	b.WriteString(StyleGameTitle.Render(gameTitle))
	b.WriteString("\n")
	b.WriteString(StyleCategory.Render(p.Category))
	b.WriteString("\n\n")

	if len(p.Splits) == 0 {
		b.WriteString(StyleHelp.Render("No splits in this preset. Press 'p' to pick or edit a preset."))
		return StyleBox.Render(b.String())
	}

	nameW := 0
	for _, s := range p.Splits {
		if len(s) > nameW {
			nameW = len(s)
		}
	}
	if nameW < 10 {
		nameW = 10
	}

	showPB := a.theme.ShowPB

	// row prefix is: 2-char marker + 1-char check + 1-char space = 4 cols,
	// so the header needs the same left offset to line up under names.
	const rowPrefix = 4
	header := strings.Repeat(" ", rowPrefix) + padRight("Split", nameW) + padLeft("Hits", 6)
	if showPB {
		header += padLeft("PB", 7)
	}

	totalHits, totalPB := 0, 0
	type rowInfo struct {
		text string
		kind int // 0=normal, 1=beatenHit, 2=beatenClear, 3=activeHit, 4=activeNoHit
	}
	rows := make([]rowInfo, len(p.Splits))
	for i, name := range p.Splits {
		st := ps.Current[i]
		pb := ps.PB[i]
		totalHits += st.Hits
		totalPB += pb.Hits

		marker := "  "
		if i == a.cursor {
			marker = "> "
		}
		check := " "
		if st.Beaten {
			if st.Hits > 0 {
				check = "\u2717"
			} else {
				check = "\u2713"
			}
		}

		row := marker + check + " " + padRight(name, nameW) + padLeft(fmtInt(st.Hits), 6)
		if showPB {
			row += padLeft(fmtInt(pb.Hits), 7)
		}

		kind := 0
		switch {
		case st.Beaten && st.Hits > 0:
			kind = 1
		case st.Beaten:
			kind = 2
		case i == a.cursor && st.Hits > 0:
			kind = 3
		case i == a.cursor:
			kind = 4
		}
		rows[i] = rowInfo{text: row, kind: kind}
	}

	totalRow := strings.Repeat(" ", rowPrefix) + padRight("Total", nameW) + padLeft(fmtInt(totalHits), 6)
	if showPB {
		totalRow += padLeft(fmtInt(totalPB), 7)
	}

	kb := a.kb
	help1 := joinHelp(
		pair(kb.Up, kb.Down)+" move",
		altPair(kb.Hit, kb.Undo)+" hit",
		label(kb.Split, "split"),
		label(kb.Unsplit, "unsplit"),
	)
	help2 := joinHelp(
		label(kb.SavePB, "save PB"),
		label(kb.DeletePB, "delete PB"),
		label(kb.Reset, "reset"),
		label(kb.Presets, "presets"),
		label(kb.Quit, "quit"),
	)

	// Pad highlighted rows a few chars past their own text, instead of
	// stopping the background right after the numbers - but don't stretch
	// them out to match the (much wider) help-text lines below.
	contentWidth := widestLine(header, totalRow)
	for _, r := range rows {
		if w := lipgloss.Width(r.text); w > contentWidth {
			contentWidth = w
		}
	}
	contentWidth += 3

	b.WriteString(StyleHeader.Render(header))
	b.WriteString("\n")

	for _, r := range rows {
		var line string
		switch r.kind {
		case 1:
			line = StyleHitRow.Render(r.text)
		case 2:
			line = StyleBeatenRow.Render(r.text)
		case 3:
			line = StyleActiveHit.Width(contentWidth).Render(r.text)
		case 4:
			line = StyleActiveNoHit.Width(contentWidth).Render(r.text)
		default:
			line = StyleNormalRow.Render(r.text)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(StyleTotal.Render(totalRow))
	b.WriteString("\n\n")
	b.WriteString(StyleHelp.Render(help1))
	b.WriteString("\n")
	b.WriteString(StyleHelp.Render(help2))

	return StyleBox.Render(b.String())
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func padLeft(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return strings.Repeat(" ", w-len(s)) + s
}
