package ipc

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestServeDoesNotStealLiveSocket(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "peeporun.sock")
	stop, err := Serve(path, func(cmd Command) {
		cmd.Result <- Result{OK: true, Msg: "alive"}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	if _, err := Serve(path, func(Command) {}); err == nil {
		t.Fatal("second Serve unexpectedly replaced the live socket")
	}
	ok, msg, err := SendCommand(path, ActionHit, "")
	if err != nil || !ok || msg != "alive" {
		t.Fatalf("original listener stopped working: ok=%v msg=%q err=%v", ok, msg, err)
	}

	// If the pathname is externally replaced, stopping the first listener
	// must not unlink the replacement endpoint.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	replacement, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = replacement.Close() }()
	stop()
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("stop removed a replacement socket: %v", err)
	}
}

func TestServeRemovesStaleSocket(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "peeporun.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	unixListener, ok := listener.(*net.UnixListener)
	if !ok {
		t.Fatalf("listener type = %T, want *net.UnixListener", listener)
	}
	unixListener.SetUnlinkOnClose(false)
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}

	stop, err := Serve(path, func(Command) {})
	if err != nil {
		t.Fatalf("Serve rejected stale socket: %v", err)
	}
	stop()
}
