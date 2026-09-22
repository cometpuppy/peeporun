package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/cometpuppy/peeporun/internal/atomicfile"
)

func DefaultPresets() []Preset {
	return []Preset{
		{
			ID:       "ds1-any",
			Game:     "Dark Souls",
			Category: "Any%",
			Splits: []string{
				"Asylum",
				"Gargoyles",
				"Quelaag",
				"Iron Golem",
				"Ornstein & Smough",
				"Pinwheel",
				"Sif",
				"Seath",
				"Nito",
				"Bed of Chaos",
				"Four Kings",
				"Gwyn",
			},
		},
		{
			ID:       "ds2-any-shulva",
			Game:     "Dark Souls II",
			Category: "Any% (Shulva)",
			Splits: []string{
				"Dragonrider",
				"Last Giant",
				"Pursuer",
				"Rotten x2",
				"Shulva",
				"Rotten x2",
				"Dragonriders",
				"Mirror Knight",
				"Demon of Song",
				"Velstadt",
				"Guardian Dragon",
				"Giant Lord",
				"Throne Watchers",
				"Nashandra",
			},
		},
		{
			ID:       "ds3-any",
			Game:     "Dark Souls III",
			Category: "Any%",
			Splits: []string{
				"Gundyr",
				"Vordt",
				"Crystal Sage",
				"Abyss Watchers",
				"Wolnir",
				"Dancer",
				"Deacons",
				"Pontiff",
				"Aldrich",
				"Yhorm",
				"Dragonslayer Armor",
				"Twin Princes",
				"Soul of Cinder",
			},
		},
	}
}

// presetFile is the on-disk shape of a single preset file. It deliberately
// has no ID field - a preset's ID is always just its filename (without
// .toml), so sharing a preset is literally just sharing this one file and
// dropping it into someone else's presets/ folder, with no internal ID to
// worry about colliding or keeping in sync with the filename.
type presetFile struct {
	Game     string   `toml:"game"`
	Category string   `toml:"category"`
	Splits   []string `toml:"splits"`
}

func presetFilePath(dir, id string) string {
	return filepath.Join(dir, id+".toml")
}

// validatePresetID rejects identifiers that cannot safely and unambiguously
// map to one filename. Replacing unsafe characters would allow two logical
// IDs to silently collide or change identity after a save/load round trip.
func validatePresetID(id string) error {
	switch {
	case id == "":
		return fmt.Errorf("preset ID cannot be empty")
	case id != strings.TrimSpace(id):
		return fmt.Errorf("preset ID %q cannot have leading or trailing whitespace", id)
	case id == "." || id == "..":
		return fmt.Errorf("preset ID %q is reserved", id)
	case filepath.Base(id) != id || strings.ContainsAny(id, `/\\`):
		return fmt.Errorf("preset ID %q cannot contain path separators", id)
	case len(id) > 128:
		return fmt.Errorf("preset ID is too long (maximum 128 bytes)")
	case strings.IndexFunc(id, unicode.IsControl) >= 0:
		return fmt.Errorf("preset ID %q cannot contain control characters", id)
	default:
		return nil
	}
}

// LoadPresets loads every preset from ~/.config/peeporun/presets/*.toml,
// one preset per file, with the preset's ID always taken from the
// filename. On first run it migrates an old single-file presets.toml if
// one exists (preserving each preset's original ID, so existing save
// data / PBs still line up correctly), or falls back to the built-in
// DS1/DS2/DS3 defaults if nothing exists yet at all.
func LoadPresets() ([]Preset, error) {
	dir, err := PresetsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".toml") {
			files = append(files, e)
		}
	}

	if len(files) == 0 {
		if migrated, err := migrateLegacyPresets(dir); err != nil {
			return nil, err
		} else if migrated != nil {
			return migrated, nil
		}
		// Truly fresh install - write the defaults as individual files.
		defaults := DefaultPresets()
		if err := SavePresets(defaults); err != nil {
			return defaults, err
		}
		return defaults, nil
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

	presets := make([]Preset, 0, len(files))
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, err
		}
		var pf presetFile
		if err := toml.Unmarshal(data, &pf); err != nil {
			return nil, err
		}
		id := strings.TrimSuffix(f.Name(), ".toml")
		if err := validatePresetID(id); err != nil {
			return nil, fmt.Errorf("invalid preset filename %q: %w", f.Name(), err)
		}
		for _, existing := range presets {
			if strings.EqualFold(existing.ID, id) {
				return nil, fmt.Errorf("preset IDs %q and %q collide", existing.ID, id)
			}
		}
		presets = append(presets, Preset{
			ID:       id,
			Game:     pf.Game,
			Category: pf.Category,
			Splits:   pf.Splits,
		})
	}
	return presets, nil
}

// migrateLegacyPresets checks for an old single-file presets.toml and, if
// found, splits it into individual files in dir (using each preset's
// existing ID as the filename, so save.toml's per-preset progress/PB data
// still matches up). The old file is renamed to presets.toml.bak rather
// than deleted, as a safety net. Returns nil (with no error) if there was
// nothing to migrate.
func migrateLegacyPresets(dir string) ([]Preset, error) {
	legacyPath, err := LegacyPresetsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(legacyPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var pf PresetsFile
	if err := toml.Unmarshal(data, &pf); err != nil {
		return nil, err
	}
	if len(pf.Presets) == 0 {
		return nil, nil
	}

	for _, p := range pf.Presets {
		if err := writePresetFile(dir, p); err != nil {
			return nil, err
		}
	}

	// Keep the old file as a backup rather than silently deleting it.
	if err := os.Rename(legacyPath, legacyPath+".bak"); err != nil {
		return nil, fmt.Errorf("back up legacy presets: %w", err)
	}

	return pf.Presets, nil
}

func writePresetFile(dir string, p Preset) error {
	if err := validatePresetID(p.ID); err != nil {
		return err
	}
	data, err := toml.Marshal(presetFile{
		Game:     p.Game,
		Category: p.Category,
		Splits:   p.Splits,
	})
	if err != nil {
		return err
	}
	return atomicfile.WriteFile(presetFilePath(dir, p.ID), data, 0o644)
}

// SavePresets writes every preset to its own file in the presets
// directory, and removes any leftover files for presets that no longer
// exist (e.g. after a deletion in the TUI).
func SavePresets(presets []Preset) error {
	dir, err := PresetsDir()
	if err != nil {
		return err
	}

	keep := make(map[string]bool, len(presets))
	for i, p := range presets {
		if err := validatePresetID(p.ID); err != nil {
			return fmt.Errorf("invalid preset %q: %w", p.ID, err)
		}
		for _, previous := range presets[:i] {
			if strings.EqualFold(previous.ID, p.ID) {
				return fmt.Errorf("preset IDs %q and %q collide", previous.ID, p.ID)
			}
		}
	}
	for _, p := range presets {
		if err := writePresetFile(dir, p); err != nil {
			return err
		}
		keep[p.ID+".toml"] = true
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		if !keep[e.Name()] {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
				return fmt.Errorf("remove obsolete preset %q: %w", e.Name(), err)
			}
		}
	}
	return nil
}
