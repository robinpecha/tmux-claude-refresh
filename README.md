# autoclaude

A TUI that watches tmux panes running [Claude Code](https://claude.com/claude-code) and automatically sends "continue" when a rate limit resets.

Fork of [henryaj/autoclaude](https://github.com/henryaj/autoclaude), updated to detect Claude Code's newer rate-limit messages (`You've hit your session limit`, `You've hit your weekly limit`, `You're out of extra usage`).

## Install

Latest release:

```bash
curl -sL https://github.com/robinpecha/tmux-claude-refresh/releases/latest/download/autoclaude_linux_amd64.tar.gz \
  | sudo tar -xz -C /usr/local/bin autoclaude
```

Pinned to a specific version (e.g. v0.1.4):

```bash
curl -sL https://github.com/robinpecha/tmux-claude-refresh/releases/download/v0.1.4/autoclaude_0.1.4_linux_amd64.tar.gz \
  | sudo tar -xz -C /usr/local/bin autoclaude
```

See all binaries (macOS, Linux, arm64) on the [Releases](https://github.com/robinpecha/tmux-claude-refresh/releases) page.

Requires `tmux`. Run it inside a tmux session:

```bash
autoclaude
```

## Usage

1. Start `autoclaude` in a tmux pane.
2. Move to a Claude Code pane with the arrow keys.
3. Press `tab` to enable auto-continue for that pane.
4. Leave it running — it sends `continue` when the rate limit resets.

### Keys

| Key | Action |
|-----|--------|
| `←↑↓→` | Navigate panes |
| `tab` | Toggle auto-continue |
| `a` | Auto-continue all Claude Code panes |
| `n` | Disable auto-continue on all panes |
| `r` | Refresh pane layout |
| `h` / `?` | Help |
| `q` | Quit |

### Pane colors

| Color | Meaning |
|-------|---------|
| Orange | Claude Code (auto-continue off) |
| Green | Claude Code (auto-continue on) |
| Red | Rate limited (waiting for reset) |
| Cyan | Selected pane |

## How it works

1. Polls tmux panes every 3 seconds.
2. Detects Claude Code by its UI patterns.
3. Parses the reset time from the rate-limit message.
4. When the time passes, sends `Escape` → `continue` → `Enter`.

## Development

```bash
go test ./...
go build
```

## License

MIT — see [LICENSE](LICENSE). Original work by [Henry Stanley](https://henrystanley.com).
