package config

import (
	"os"
	"regexp"

	"github.com/BurntSushi/toml"
	"github.com/cometpuppy/peeporun/internal/atomicfile"
)

const DefaultAccentColor = "#FFD400"

// NamedColor is a quick-pick color for the in-TUI theme picker.
type NamedColor struct {
	Name  string // display name, e.g. "Red"
	Color string // "#RRGGBB" hex
}

// ThemePresets are the named colors available in the theme picker.
var ThemePresets = []NamedColor{
	{"Red", "#ff5f5f"},
	{"Orange", "#ff8800"},
	{"Yellow", "#FFD400"},
	{"Green", "#5fd75f"},
	{"Blue", "#4488ff"},
	{"Light Blue", "#5fafff"},
	{"Pink", "#ff5faf"},
	{"Purple", "#af5fff"},
}

var hexColorRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// ValidHexColor reports whether s is a "#RRGGBB" hex color string.
func ValidHexColor(s string) bool {
	return hexColorRe.MatchString(s)
}

func DefaultTheme() ThemeSettings {
	return ThemeSettings{AccentColor: DefaultAccentColor, ShowPB: true}
}

// LoadTheme reads theme.toml, creating it with the defaults on first run.
// If the configured color isn't a valid "#RRGGBB" hex string, it falls
// back to the default rather than failing to start. If show_pb is absent
// from an older theme.toml (from before this setting existed), it
// defaults to true rather than Go's normal bool zero-value, so upgrading
// doesn't silently hide PB for existing users.
func LoadTheme() (ThemeSettings, error) {
	path, err := ThemePath()
	if err != nil {
		return ThemeSettings{}, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		defaults := DefaultTheme()
		if werr := SaveTheme(defaults); werr != nil {
			return defaults, werr
		}
		return defaults, nil
	}
	if err != nil {
		return ThemeSettings{}, err
	}

	var raw map[string]interface{}
	if err := toml.Unmarshal(data, &raw); err != nil {
		return ThemeSettings{}, err
	}
	_, hasShowPB := raw["show_pb"]

	var t ThemeSettings
	if err := toml.Unmarshal(data, &t); err != nil {
		return ThemeSettings{}, err
	}
	if !ValidHexColor(t.AccentColor) {
		t.AccentColor = DefaultAccentColor
	}
	if !hasShowPB {
		t.ShowPB = true
	}
	return t, nil
}

func SaveTheme(t ThemeSettings) error {
	path, err := ThemePath()
	if err != nil {
		return err
	}
	data, err := toml.Marshal(t)
	if err != nil {
		return err
	}
	header := "" +
		"# The standard theme color is " + DefaultAccentColor + "\n" +
		"# show_pb controls whether Personal Best is shown, in BOTH the TUI\n" +
		"# and the OBS overlay - set to false to hide it in both places.\n"
	return atomicfile.WriteFile(path, append([]byte(header), data...), 0o644)
}
