package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cometpuppy/peeporun/internal/config"
	"github.com/cometpuppy/peeporun/internal/ipc"
	"github.com/cometpuppy/peeporun/internal/overlay"
)

type screen int

const (
	screenTracker screen = iota
	screenPresetSelect
	screenPresetEdit
	screenThemePicker
)

type confirmKind int

const (
	confirmNone confirmKind = iota
	confirmReset
	confirmSavePB
	confirmDeletePreset
	confirmDeleteSplit
	confirmDeletePB
)

// App is the root Bubble Tea model.
type App struct {
	kb      config.Keybinds
	presets []config.Preset
	save    config.SaveFile
	overlay config.OverlaySettings
	theme   config.ThemeSettings

	presetIdx int // index into presets, currently loaded in tracker
	cursor    int // active split index in tracker

	screen  screen
	confirm confirmKind

	presetLoaded bool // true once the user has explicitly chosen a preset

	width, height  int
	status         string
	persistenceErr error
	presetsDirty   bool

	// preset select screen
	selCursor int

	// preset edit screen
	editPresetIdx int
	editCursor    int // -2 = game field, -1 = category field, 0..n-1 = split index, n = "add split" row
	editMode      bool
	editAction    string // "game", "category", "rename-split", "add-split"

	// theme picker screen
	themeCursor   int
	themeEditMode bool // true when typing a custom hex code
	input         textinput.Model

	// autoNamedIDs tracks preset IDs still eligible for maybeAutoRenamePresetID
	// to keep following Game/Category as they're typed - covers both the
	// initial "new-preset" placeholder and any ID that itself came from a
	// previous auto-rename, so setting Game then Category in the same
	// session chains correctly instead of only renaming once.
	autoNamedIDs map[string]bool
}

func NewApp(kb config.Keybinds, presets []config.Preset, save config.SaveFile, overlaySettings config.OverlaySettings, theme config.ThemeSettings) *App {
	SetAccentColor(theme.AccentColor)

	ti := textinput.New()
	ti.CharLimit = 64
	ti.Width = 40

	a := &App{
		kb:           kb,
		presets:      presets,
		save:         save,
		overlay:      overlaySettings,
		theme:        theme,
		screen:       screenPresetSelect,
		input:        ti,
		autoNamedIDs: map[string]bool{},
	}

	// resume last preset if we have one
	for i, p := range presets {
		if p.ID == save.LastPreset {
			a.presetIdx = i
			a.presetLoaded = true
			break
		}
	}
	a.syncSaveShape()
	a.cursor = a.firstUnbeatenOrZero()
	a.selCursor = a.presetIdx
	if err := a.writeOverlay(); err != nil {
		a.setPersistenceError(err)
	}
	return a
}

func (a *App) Init() tea.Cmd {
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a, nil
	case ipc.Command:
		a.handleIPCCommand(msg)
		return a, nil
	case tea.KeyMsg:
		key := msg.String()

		// Clear any pending status message on the next keypress.
		if a.status != "" {
			a.status = ""
		}

		// global quit (only when not typing in a text field)
		if !a.inputActive() && config.Matches(key, a.kb.Quit) {
			if a.presetsDirty {
				a.persistPresets()
			} else {
				a.persistState()
			}
			if a.persistenceErr != nil {
				return a, nil
			}
			return a, tea.Quit
		}

		if a.confirm != confirmNone {
			return a.updateConfirm(key)
		}

		switch a.screen {
		case screenTracker:
			return a.updateTracker(key)
		case screenPresetSelect:
			return a.updatePresetSelect(key)
		case screenPresetEdit:
			return a.updatePresetEdit(msg)
		case screenThemePicker:
			return a.updateThemePicker(msg)
		}
	}
	return a, nil
}

func (a *App) inputActive() bool {
	return a.screen == screenPresetEdit && a.editMode
}

func (a *App) View() string {
	var body string
	switch a.screen {
	case screenTracker:
		body = a.viewTracker()
	case screenPresetSelect:
		body = a.viewPresetSelect()
	case screenPresetEdit:
		body = a.viewPresetEdit()
	case screenThemePicker:
		body = a.viewThemePicker()
	}

	if a.confirm != confirmNone {
		body = a.viewConfirmOverlay(body)
	}

	title := "peepoRun"
	if a.screen == screenPresetSelect {
		title = "peepoRun - A TUI Based Hit* Counter"
	}
	full := StyleAppTitle.Render(title) + "\n\n" + body
	if a.status != "" {
		full += "\n\n" + StyleHelp.Render(a.status)
	}

	if a.width == 0 {
		return full
	}
	return centerBlock(full, a.width, a.height)
}

func centerBlock(s string, w, h int) string {
	return lipglossPlace(s, w, h)
}

func (a *App) currentPreset() config.Preset {
	if len(a.presets) == 0 {
		return config.Preset{Game: "No Presets", Category: "Press p", Splits: nil}
	}
	return a.presets[a.presetIdx]
}

// syncSaveShape ensures the save entry for the currently loaded tracker
// preset has state slices matching its current split count.
func (a *App) syncSaveShape() {
	if len(a.presets) == 0 {
		return
	}
	a.syncSaveShapeFor(a.currentPreset())
}

// syncSaveShapeFor does the same thing but for an arbitrary preset -
// needed when editing a preset that isn't necessarily the one currently
// loaded in the tracker (e.g. editing preset B while preset A is active).
func (a *App) syncSaveShapeFor(p config.Preset) {
	ps := a.save.Presets[p.ID]

	for len(ps.Current) < len(p.Splits) {
		ps.Current = append(ps.Current, config.SplitState{})
	}
	ps.Current = ps.Current[:len(p.Splits)]

	for len(ps.PB) < len(p.Splits) {
		ps.PB = append(ps.PB, config.PBEntry{})
	}
	ps.PB = ps.PB[:len(p.Splits)]

	if a.save.Presets == nil {
		a.save.Presets = map[string]config.PresetSave{}
	}
	a.save.Presets[p.ID] = ps
}

func (a *App) firstUnbeatenOrZero() int {
	p := a.currentPreset()
	ps := a.save.Presets[p.ID]
	for i, s := range ps.Current {
		if !s.Beaten {
			return i
		}
	}
	if len(p.Splits) == 0 {
		return 0
	}
	return len(p.Splits) - 1
}

// handleIPCCommand executes a command received from `peeporun hit`,
// `peeporun split`, etc. (see internal/ipc) and reports the outcome back
// on cmd.Result exactly once. This runs on the same goroutine as normal
// keypress handling, so it's safe to touch app state directly here -
// that's the whole point of routing external commands through the Bubble
// Tea message loop instead of handling them on their own goroutine.
func (a *App) handleIPCCommand(cmd ipc.Command) {
	res := ipc.Result{}
	defer func() { cmd.Result <- res }()

	noPreset := func() {
		res.OK = false
		res.Msg = "no preset loaded - open peepoRun and select one first"
	}
	finish := func() {
		if a.persistenceErr != nil {
			res.OK = false
			res.Msg = a.persistenceErr.Error()
			return
		}
		res.OK = true
	}
	a.persistenceErr = nil

	switch cmd.Action {
	case ipc.ActionHit:
		if len(a.presets) == 0 || !a.presetLoaded {
			noPreset()
			return
		}
		a.addHit(1)
		finish()
	case ipc.ActionUndo:
		if len(a.presets) == 0 || !a.presetLoaded {
			noPreset()
			return
		}
		a.addHit(-1)
		finish()
	case ipc.ActionSplit:
		if len(a.presets) == 0 || !a.presetLoaded {
			noPreset()
			return
		}
		a.beatCurrentAndAdvance()
		finish()
	case ipc.ActionUnsplit:
		if len(a.presets) == 0 || !a.presetLoaded {
			noPreset()
			return
		}
		a.unsplit()
		finish()
	case ipc.ActionReset:
		// Deliberately skips the confirmation dialog the R key shows in
		// the TUI - this path is meant for hotkey/CLI use while tabbed
		// into the game, where there's no way to interact with a dialog
		// sitting in a terminal you're not looking at.
		if len(a.presets) == 0 || !a.presetLoaded {
			noPreset()
			return
		}
		a.resetRun()
		finish()
	case ipc.ActionPreset:
		if cmd.Arg == "" {
			res.OK = false
			res.Msg = "usage: peeporun preset <id>"
			return
		}
		idx := -1
		for i, p := range a.presets {
			if strings.EqualFold(p.ID, cmd.Arg) {
				idx = i
				break
			}
		}
		if idx == -1 {
			res.OK = false
			res.Msg = "no such preset: " + cmd.Arg
			return
		}
		a.presetIdx = idx
		a.presetLoaded = true
		a.syncSaveShape()
		a.cursor = a.firstUnbeatenOrZero()
		a.screen = screenTracker
		a.persistState()
		finish()
		if res.OK {
			res.Msg = "switched to " + a.presets[idx].ID
		}
	default:
		res.OK = false
		res.Msg = "unknown command: " + string(cmd.Action)
	}
}

func (a *App) persistState() {
	a.persistenceErr = nil
	if len(a.presets) > 0 {
		a.save.LastPreset = a.currentPreset().ID
	}
	if err := config.SaveState(a.save); err != nil {
		a.setPersistenceError(err)
		return
	}
	if err := a.writeOverlay(); err != nil {
		a.setPersistenceError(err)
		return
	}
}

func (a *App) persistPresets() {
	a.persistenceErr = nil
	a.presetsDirty = true
	if err := config.SavePresets(a.presets); err != nil {
		a.setPersistenceError(err)
		return
	}
	a.presetsDirty = false
	a.persistState()
}

func (a *App) setPersistenceError(err error) {
	a.persistenceErr = err
	a.status = "Save failed: " + err.Error()
}

func (a *App) writeOverlay() error {
	data := overlay.Data{ShowPB: a.theme.ShowPB}
	if len(a.presets) > 0 && a.presetLoaded {
		p := a.currentPreset()
		ps := a.save.Presets[p.ID]
		data.Game = p.Game
		data.Category = p.Category
		for i, name := range p.Splits {
			st := ps.Current[i]
			pb := ps.PB[i]
			data.Splits = append(data.Splits, overlay.Split{
				Name:   name,
				Hits:   st.Hits,
				PB:     pb.Hits,
				Beaten: st.Beaten,
				Active: i == a.cursor,
			})
			data.TotalHits += st.Hits
			data.TotalPB += pb.Hits
		}
	}
	return overlay.Write(data, a.overlay, a.theme.AccentColor)
}

func fmtInt(n int) string {
	return fmt.Sprintf("%d", n)
}
