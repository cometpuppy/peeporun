// Package ipc lets a short-lived `peeporun <command>` invocation control an
// already-running peepoRun TUI instance, over a local Unix domain socket.
//
// This is what makes external hotkey binding (e.g. KDE Custom Shortcuts
// running `peeporun split`) possible without any OS-level global key
// capture - the desktop environment's own trusted shortcut system runs a
// normal command, which just talks to the real running instance over this
// socket. Works identically under X11 and Wayland, since it never touches
// input devices or window-system APIs at all.
package ipc

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Action is a command understood by the running application.
type Action string

const (
	ActionHit     Action = "hit"
	ActionUndo    Action = "undo"
	ActionSplit   Action = "split"
	ActionUnsplit Action = "unsplit"
	ActionReset   Action = "reset"
	ActionPreset  Action = "preset"
)

// IsAction reports whether s is a supported IPC action.
func IsAction(s string) bool {
	switch Action(s) {
	case ActionHit, ActionUndo, ActionSplit, ActionUnsplit, ActionReset, ActionPreset:
		return true
	default:
		return false
	}
}

// Command is one request from a client (`peeporun hit`, etc.) to the
// running TUI instance. Action is one of "hit", "undo", "split", "unsplit", "reset",
// "preset" (Arg is the preset ID for that last one). Result must be sent
// on exactly once by whoever handles the command.
type Command struct {
	Action Action
	Arg    string
	Result chan Result
}

// Result is the outcome of handling a Command, sent back to the waiting
// client connection.
type Result struct {
	OK  bool
	Msg string
}

// Serve starts listening on socketPath and, for every line received on a
// connection, builds a Command (with a fresh Result channel) and passes it
// to handle. handle must eventually send exactly one Result on cmd.Result -
// typically by routing the Command into a tea.Program via p.Send(cmd) and
// having the Bubble Tea Update() loop reply on the channel once it's
// processed the action, since that's the only goroutine allowed to touch
// the app's state.
//
// Returns a stop func to shut the listener down and remove the socket
// file - call it on program exit.
func Serve(socketPath string, handle func(Command)) (stop func(), err error) {
	if err := removeStaleSocket(socketPath); err != nil {
		return nil, err
	}

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", socketPath, err)
	}
	unixListener, ok := ln.(*net.UnixListener)
	if !ok {
		_ = ln.Close()
		return nil, fmt.Errorf("listen on %s returned unexpected listener type %T", socketPath, ln)
	}
	// net.UnixListener otherwise unlinks the current pathname on Close,
	// even if another endpoint has replaced it. Cleanup below verifies file
	// identity before removing the socket we created.
	unixListener.SetUnlinkOnClose(false)
	if err := os.Chmod(socketPath, 0o600); err != nil {
		_ = ln.Close()
		_ = os.Remove(socketPath)
		return nil, fmt.Errorf("secure socket %s: %w", socketPath, err)
	}
	socketInfo, err := os.Lstat(socketPath)
	if err != nil {
		_ = ln.Close()
		_ = os.Remove(socketPath)
		return nil, fmt.Errorf("inspect listening socket %s: %w", socketPath, err)
	}

	done := make(chan struct{})
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-done:
					return // listener closed intentionally by stop()
				default:
					continue
				}
			}
			go serveConn(conn, handle)
		}
	}()

	var stopOnce sync.Once
	stop = func() {
		stopOnce.Do(func() {
			close(done)
			_ = ln.Close()
			currentInfo, statErr := os.Lstat(socketPath)
			if statErr == nil && os.SameFile(socketInfo, currentInfo) {
				_ = os.Remove(socketPath)
			}
		})
	}
	return stop, nil
}

func removeStaleSocket(socketPath string) error {
	info, err := os.Lstat(socketPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect socket %s: %w", socketPath, err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("refusing to remove non-socket path %s", socketPath)
	}

	conn, dialErr := net.DialTimeout("unix", socketPath, 250*time.Millisecond)
	if dialErr == nil {
		_ = conn.Close()
		return fmt.Errorf("another peepoRun instance is already listening on %s", socketPath)
	}
	if !errors.Is(dialErr, syscall.ECONNREFUSED) && !errors.Is(dialErr, os.ErrNotExist) {
		return fmt.Errorf("cannot determine whether socket %s is stale: %w", socketPath, dialErr)
	}
	if err := os.Remove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale socket %s: %w", socketPath, err)
	}
	return nil
}

func serveConn(conn net.Conn, handle func(Command)) {
	defer func() { _ = conn.Close() }()
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return
	}
	action, arg, _ := strings.Cut(strings.TrimSpace(scanner.Text()), " ")
	if action == "" {
		_, _ = fmt.Fprintln(conn, "ERR empty command")
		return
	}

	resultCh := make(chan Result, 1)
	handle(Command{Action: Action(action), Arg: arg, Result: resultCh})

	select {
	case res := <-resultCh:
		if res.OK {
			_, _ = fmt.Fprintln(conn, strings.TrimSpace("OK "+res.Msg))
		} else {
			_, _ = fmt.Fprintln(conn, strings.TrimSpace("ERR "+res.Msg))
		}
	case <-time.After(3 * time.Second):
		_, _ = fmt.Fprintln(conn, "ERR timed out waiting for the running instance to respond")
	}
}

// SendCommand is the client side: connects to socketPath, sends action
// (plus arg, if any), and waits for the response. A connection failure
// means no peepoRun instance is currently listening there (not running,
// or the socket path is wrong) - the caller should treat that as "not
// running" rather than a generic error.
func SendCommand(socketPath string, action Action, arg string) (ok bool, msg string, err error) {
	conn, err := net.DialTimeout("unix", socketPath, 1*time.Second)
	if err != nil {
		return false, "", err
	}
	defer func() { _ = conn.Close() }()

	line := string(action)
	if arg != "" {
		line += " " + arg
	}
	if _, err := fmt.Fprintln(conn, line); err != nil {
		return false, "", err
	}

	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return false, "", err
	}
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return false, "", fmt.Errorf("no response from running instance")
	}
	resp := scanner.Text()
	switch {
	case strings.HasPrefix(resp, "OK"):
		return true, strings.TrimSpace(strings.TrimPrefix(resp, "OK")), nil
	case strings.HasPrefix(resp, "ERR"):
		return false, strings.TrimSpace(strings.TrimPrefix(resp, "ERR")), nil
	default:
		return false, "", fmt.Errorf("unexpected response: %q", resp)
	}
}
