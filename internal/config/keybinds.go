package config

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/cometpuppy/peeporun/internal/atomicfile"
)

func DefaultKeybinds() Keybinds {
	return Keybinds{
		Up:          []string{"up", "k"},
		Down:        []string{"down", "j"},
		Hit:         []string{"+", "="},
		Undo:        []string{"-", "_"},
		Split:       []string{" ", "space"},
		Unsplit:     []string{"u"},
		SavePB:      []string{"S"},
		DeletePB:    []string{"D"},
		Reset:       []string{"R"},
		Presets:     []string{"p"},
		Quit:        []string{"q", "ctrl+c"},
		Confirm:     []string{"enter", "y"},
		Cancel:      []string{"esc"},
		New:         []string{"n"},
		Edit:        []string{"e"},
		Delete:      []string{"d"},
		Add:         []string{"a"},
		Rename:      []string{"r"},
		MoveUp:      []string{"K", "shift+up"},
		MoveDn:      []string{"J", "shift+down"},
		ThemePicker: []string{"t"},
	}
}

// LoadKeybinds reads keys.toml, creating it with defaults on first run.
func LoadKeybinds() (Keybinds, error) {
	path, err := KeysPath()
	if err != nil {
		return Keybinds{}, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		defaults := DefaultKeybinds()
		if werr := SaveKeybinds(defaults); werr != nil {
			return defaults, werr
		}
		return defaults, nil
	}
	if err != nil {
		return Keybinds{}, err
	}

	var kb Keybinds
	if err := toml.Unmarshal(data, &kb); err != nil {
		return Keybinds{}, err
	}
	return kb, nil
}

func SaveKeybinds(kb Keybinds) error {
	path, err := KeysPath()
	if err != nil {
		return err
	}
	data, err := toml.Marshal(kb)
	if err != nil {
		return err
	}
	return atomicfile.WriteFile(path, data, 0o644)
}
