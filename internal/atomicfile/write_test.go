package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileReplacesCompleteFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "state.toml")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, []byte("new contents"), 0o640); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new contents" {
		t.Fatalf("got %q, want complete replacement", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if gotPerm := info.Mode().Perm(); gotPerm != 0o640 {
		t.Fatalf("permissions = %o, want 640", gotPerm)
	}
}
