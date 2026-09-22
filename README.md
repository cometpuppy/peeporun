# peepoRun

**peepoRun** is a TUI Hit Counter.

## Features

- Track your Hits over the Course of a whole Run
- HTML Overlay for your Stream via `overlay.html`
- Customizable Splits / Presets in the TUI or via `.toml files`
- Customizable Keybindings via `keys.toml`
- Customizable Theme color with Hex Codes via `theme.toml` or TUI Theme Picker

## Installation

## Linux

### Build from Source

Requires Go 1.22+.

```bash
git clone https://github.com/cometpuppy/peeporun.git
cd peeporun
go build -o peeporun .
./peeporun
```

### Using the Prebuilt Binary

Download the peeporun_linux_amd64/arm64.tar.gz from the Releases Page

```bash
tar -xzf peeporun_linux_amd64.tar.gz
./peeporun
```

### Using Go Install

Requires Go 1.22+.

```bash
go install github.com/cometpuppy/peeporun@latest
```

### With Brew

```bash
brew install cometpuppy/peeporun/peeporun
```

Now just run `peeporun` from your Terminal.

## Windows

### With Scoop

```bash
scoop bucket add peeporun https://github.com/cometpuppy/scoop-peeporun
scoop install peeporun
```

Now just run `peeporun`from your Terminal.

### Using the Prebuilt Binary

Download the peeporun_windows_amd64.zip from the Releases Page
Extract it and run the peeporun.exe

*It will show a "Windows protected your PC" prompt — this is because peepoRun is not code-signed, not because it's harmful. Click "More info" → "Run anyway".*

## Usage

Run `peeporun` to launch the TUI.

```bash
Usage: peeporun [COMMAND] [ARGS]

Commands:
  hit             Add a hit to the current split of an already-running instance
  undo            Remove a hit from the current split
  split           Mark the current split beaten and advance
  unsplit         Un-beat the current split (or step back to the previous one)
  reset           Reset the current run to 0 hits (no confirmation)
  preset <id>     Switch to a different preset by ID
  help            Print this help message

Options:
  -v, --version   Print version and exit
  -h, --help      Print this help message
```

## First Launch

On first launch, peepoRun creates (if missing):

| File                              | Purpose                                                             |
| --------------------------------- | ------------------------------------------------------------------- |
| `~/.config/peeporun/keys.toml`    | Keybindings                                                         |
| `~/.config/peeporun/presets/`     | Your presets, one `.toml` file each (DS1 / DS2 / DS3 Any% built in) |
| `~/.config/peeporun/save.toml`    | Current run + PB per preset                                         |
| `~/.config/peeporun/overlay.html` | Overlay for Streaming                                               |
| `~/.config/peeporun/overlay.toml` | Turn the Overlay off or on                                          |
| `~/.config/peeporun/theme.toml`   | Change the Accent Color of the TUI and Overlay                      |

(On Windows: `%AppData%\\peeporun\\`)

## Default keybinds

Inside the Preset:

| Key              | Action                                                            |
| ---------------- | ----------------------------------------------------------------- |
| `↑`/`k`, `↓`/`j` | Move cursor between splits                                        |
| `+` / `=`        | Add a hit to the split under the cursor                           |
| `-` / `_`        | Remove a hit from the split under the cursor                      |
| `Space`          | Mark the current split "beaten" and advance to the next one       |
| `u`              | Unsplit - undo the last "beaten" mark and move back               |
| `S`              | Manually save the current run as new Personal Best                |
| `D`              | Delete the current preset's Personal Best (confirmation required) |
| `R`              | Reset the current run to 0 hits (confirmation required)           |
| `p`              | Open the preset select screen                                     |
| `q` / `Ctrl+C`   | Quit                                                              |

Inside the preset select screen:

| Key              | Action                                         |
| ---------------- | ---------------------------------------------- |
| `↑`/`k`, `↓`/`j` | Move selection                                 |
| `Enter`          | Load selected preset into the tracker          |
| `e`              | Edit selected preset                           |
| `n`              | Create a new (empty) preset                    |
| `d`              | Delete selected preset (confirmation required) |
| `t`              | Open the theme color picker                    |
| `Esc`            | Back to tracker                                |

Inside the preset editor:

| Key                                | Action                                                  |
| ---------------------------------- | ------------------------------------------------------- |
| `↑`/`k`, `↓`/`j`                   | Move between Game / Category / each split / "Add split" |
| `Enter` or `r`                     | Rename the selected field                               |
| `a`                                | Add a new split                                         |
| `d`                                | Delete the selected split (confirmation required)       |
| `K` / `J` or `Shift+↑` / `Shift+↓` | Move the selected split up / down                       |
| `Esc`                              | Back to preset select                                   |

## How the Hit Counter works

- `+`/`-` change the hit count of whichever split the cursor is on.
- `Space` locks in the current split as beaten and moves you to the next one.
  `u` undoes that: it un-marks the split as beaten and moves the
  cursor back to it (or, if you're on the last split that hasn't advanced
  yet, just un-marks it in place).
- `S`  opens the "save as PB?" confirmation manually
- `D` clears the current preset's PB entirely
- `R` resets the current run's hits/progress back to 0
- A beaten split shows green with a checkmark if you took **0 hits** on
  it, or red with an ✗ if you took **1 or more hits**.
- When the **last** split in a preset is beaten, if the run's total hits is
  lower than the saved PB (or there's no PB yet), you'll be asked to save
  it as the new Personal Best.

## Built-in presets

These Presets are the Splits I personally use for DS1 - DS3.

- **Dark Souls — Any%**: Asylum → Gargoyles → Quelaag → Iron Golem →
  Ornstein & Smough → Pinwheel → Sif → Seath → Nito → Bed of Chaos →
  Four Kings → Gwyn
- **Dark Souls II — Any% (Shulva)**: Dragonrider → Last Giant → Pursuer →
  Rotten x2 → Shulva → Rotten x2 → Dragonriders → Mirror Knight →
  Demon of Song → Velstadt → Guardian Dragon → Giant Lord →
  Throne Watchers → Nashandra
- **Dark Souls III — Any%**: Gundyr → Vordt → Crystal Sage →
  Abyss Watchers → Wolnir → Dancer → Deacons → Pontiff → Aldrich →
  Yhorm → Dragonslayer Armor → Twin Princes → Soul of Cinder

Add your own via the TUI editor, or by hand-editing / dropping in files
under `~/.config/peeporun/presets/`. Each preset is its own file (e.g.
`ds3-any.toml`), named after the preset it represents. The filename stem is
the preset ID; unsafe IDs (empty/dot names, path separators, surrounding
whitespace, or case-insensitive duplicates) are rejected rather than silently
rewritten to a different ID.

```toml
game = "Dark Souls III"
category = "Any%"
splits = ["Gundyr", "Vordt", "Crystal Sage", # ...
]
```

## Using peepoRun as an OBS overlay

peepoRun automatically writes a live-updating HTML file to
`~/.config/peeporun/overlay.html` 
(`%AppData%\\peeporun\\overlay.html` on Windows),  mirroring exactly what the tracker screen shows, Game,
Category, Every Split's Hits and PB.

To show it on stream (OBS):

1. In OBS, add a new **Browser Source**.
2. Check **Local file**, and point it at `overlay.html` from the path above.
3. Set a custom width/height if you please

**Settings:** `~/.config/peeporun/overlay.toml` controls it:

```toml
enabled = true
refresh_seconds = 1.5
```

Set `enabled = false` to stop writing the file entirely, or tune
`refresh_seconds` to make it refresh faster or slower. No preset is
shown in the overlay until you've actually selected one from the preset
menu, it stays blank until then.

## Changing the accent color

### Per File

`~/.config/peeporun/theme.toml` (`%AppData%\\peeporun\\theme.toml` on
Windows) controls the accent color used across both the TUI and the OBS
overlay, titles, borders, the "active row" highlight, the Total row,
all of it:

```toml
accent_color = "#FFD400"
```

Change the hex code, save, and relaunch peepoRun.

`theme.toml` also has a `show_pb` setting:

```toml
show_pb = true
```

Set it to `false` to hide the PB column entirely — in **both** the TUI
and the overlay at once, since it's one shared setting rather than two
separate ones. Useful if you only care about hits for a given run and
don't want the PB comparison cluttering the view (or your stream).

### Theme Picker (in-TUI)

Press `t` in the preset menu to open the **Theme Picker**:

- **Named colors**: Red, Orange, Yellow (default), Green, Blue, Light Blue,
  Pink, Purple — navigate with `↑`/`↓` and press `Enter` to apply.
- **Custom hex**: Navigate to "Custom (#RRGGBB)" and press
  `Enter` to enter text-input mode, then type any `#RRGGBX` hex color code.
- **Show PB**: Toggle the PB column on/off in both the TUI and the
  overlay, this is the same `show_pb` setting in `theme.toml`.
- **Back**: `Esc` returns to the preset select screen without changes.
- Changes apply **instantly** to both the TUI and the OBS `overlay.html` —
  no restart needed. If the value
  isn't a valid `#RRGGBB` hex color, peepoRun falls back to the
  default yellow.
