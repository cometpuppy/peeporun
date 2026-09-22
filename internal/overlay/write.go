package overlay

import (
	"github.com/cometpuppy/peeporun/internal/atomicfile"
	"github.com/cometpuppy/peeporun/internal/config"
)

// Write renders the given data and writes it to the configured overlay.html
// path, if the overlay is enabled. Safe to call frequently (e.g. after every
// tracker action) since it's just a local file write.
func Write(d Data, settings config.OverlaySettings, accent string) error {
	if !settings.Enabled {
		return nil
	}
	path, err := config.OverlayHTMLPath()
	if err != nil {
		return err
	}
	html := Render(d, settings.RefreshSeconds, accent)
	return atomicfile.WriteFile(path, []byte(html), 0o644)
}
