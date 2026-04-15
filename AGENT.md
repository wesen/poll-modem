# Agent Guidelines for poll-modem

## Build Commands

- Build: `go build ./...` or `go build -o poll-modem ./cmd/poll-modem`
- Test: `go test ./...`
- Run: `./poll-modem --url http://192.168.0.1 --username admin --password password`
- Lint: `make lint`
- Release snapshot: `make goreleaser`

## Project Structure

- `cmd/poll-modem/`: CLI entry point
- `cmd/root.go`: Cobra command definitions
- `internal/modem/`: HTTP client and parser for modem data
  - `client.go`: HTTP client with cookie/auth support
  - `types.go`: Data structures for modem info
  - `database.go`: SQLite persistence
- `internal/tui/`: Bubbletea TUI components

## Development Notes

- Uses charmbracelet/bubbletea for the TUI
- SQLite database stored at `~/.config/poll-modem/history.db`
- Supports Technicolor CGM4331COM-style modems (Xfinity/Cox)
- Cookie persistence for modem authentication

## Testing with tmux

When testing the TUI, use tmux:
```bash
tmux new-session -d -s poll-modem "./poll-modem --username admin --password pass"
tmux capture-pane -t poll-modem -p
tmux send-keys -t poll-modem Tab  # switch views
tmux kill-session -t poll-modem
```
