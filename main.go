package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cometpuppy/peeporun/internal/config"
	"github.com/cometpuppy/peeporun/internal/ipc"
	"github.com/cometpuppy/peeporun/internal/ui"
)

// version is set at build time via -ldflags "-X main.version=..."
// (see .goreleaser.toml). Defaults to "dev" for local `go build`.
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch {
		case os.Args[1] == "--version" || os.Args[1] == "-v":
			fmt.Println("peepoRun " + version)
			return
		case os.Args[1] == "help" || os.Args[1] == "-h" || os.Args[1] == "--help":
			printHelp()
			return
		case ipc.IsAction(os.Args[1]):
			arg := ""
			if len(os.Args) > 2 {
				arg = os.Args[2]
			}
			runClientCommand(os.Args[1], arg)
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown command: %s\nRun 'peeporun help' to see available commands.\n", os.Args[1])
			os.Exit(1)
		}
	}

	runTUI()
}

func printHelp() {
	fmt.Print(`A TUI Hit Counter for Dark Souls*

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
`)
}

// runClientCommand sends a single command to an already-running peepoRun
// instance and exits - this is the code path hit when you run e.g.
// `peeporun split` from a hotkey or script, rather than the TUI itself.
func runClientCommand(action, arg string) {
	if action == "preset" && arg == "" {
		fmt.Fprintln(os.Stderr, "usage: peeporun preset <id>")
		os.Exit(1)
	}

	socketPath := config.SocketPath()
	ok, msg, err := ipc.SendCommand(socketPath, ipc.Action(action), arg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "peepoRun doesn't seem to be running:", err)
		os.Exit(1)
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "error:", msg)
		os.Exit(1)
	}
	if msg != "" {
		fmt.Println(msg)
	}
}

func runTUI() {
	kb, err := config.LoadKeybinds()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load keybinds:", err)
		os.Exit(1)
	}

	presets, err := config.LoadPresets()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load presets:", err)
		os.Exit(1)
	}

	save, err := config.LoadSave()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load save data:", err)
		os.Exit(1)
	}

	overlaySettings, err := config.LoadOverlaySettings()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load overlay settings:", err)
		os.Exit(1)
	}

	theme, err := config.LoadTheme()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load theme:", err)
		os.Exit(1)
	}

	app := ui.NewApp(kb, presets, save, overlaySettings, theme)

	p := tea.NewProgram(app, tea.WithAltScreen())

	stopIPC, ipcErr := ipc.Serve(config.SocketPath(), func(cmd ipc.Command) { p.Send(cmd) })
	if ipcErr != nil {
		// Not fatal - peepoRun still works fine as a plain TUI, you just
		// won't be able to control it via `peeporun split` etc. from
		// outside while this instance is running.
		fmt.Fprintln(os.Stderr, "warning: hotkey/CLI control unavailable:", ipcErr)
	} else {
		defer stopIPC()
	}

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
