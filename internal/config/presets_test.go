package config

import (
	"strings"
	"testing"
)

func TestValidatePresetID(t *testing.T) {
	t.Parallel()

	valid := []string{"ds1-any", "日本語", "challenge!", "a.b"}
	for _, id := range valid {
		if err := validatePresetID(id); err != nil {
			t.Errorf("validatePresetID(%q): %v", id, err)
		}
	}

	invalid := []string{"", " ", ".", "..", "/", `\\`, "a/b", `a\\b`, " padded ", strings.Repeat("x", 129)}
	for _, id := range invalid {
		if err := validatePresetID(id); err == nil {
			t.Errorf("validatePresetID(%q) unexpectedly succeeded", id)
		}
	}
}

func TestSavePresetsRejectsCaseInsensitiveCollision(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := SavePresets([]Preset{{ID: "Run"}, {ID: "run"}})
	if err == nil {
		t.Fatal("SavePresets accepted colliding IDs")
	}
}

func TestPresetIDSurvivesSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	want := []Preset{{ID: "日本語!", Game: "Game", Category: "Any%", Splits: []string{"One"}}}
	if err := SavePresets(want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != want[0].ID {
		t.Fatalf("loaded preset IDs = %#v, want %q", got, want[0].ID)
	}
}
