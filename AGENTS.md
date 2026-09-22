# AGENTS.md

- Keep changes small and preserve existing TUI behavior unless fixing a confirmed bug.
- Run `gofmt -w .`, `go vet ./...`, `go test ./...`, and `golangci-lint run` before submitting.
- Propagate or surface persistence and IPC errors; never silently discard them.
- Use atomic writes for user configuration/state files.
- Add focused tests for real failure paths; do not add coverage-only test bulk.
