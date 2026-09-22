package config

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/cometpuppy/peeporun/internal/atomicfile"
)

func LoadSave() (SaveFile, error) {
	path, err := SavePath()
	if err != nil {
		return SaveFile{}, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return SaveFile{Presets: map[string]PresetSave{}}, nil
	}
	if err != nil {
		return SaveFile{}, err
	}

	var sf SaveFile
	if err := toml.Unmarshal(data, &sf); err != nil {
		return SaveFile{}, err
	}
	if sf.Presets == nil {
		sf.Presets = map[string]PresetSave{}
	}
	return sf, nil
}

func SaveState(sf SaveFile) error {
	path, err := SavePath()
	if err != nil {
		return err
	}
	data, err := toml.Marshal(sf)
	if err != nil {
		return err
	}
	return atomicfile.WriteFile(path, data, 0o644)
}
