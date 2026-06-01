<div align="center">

# 🧋 `ServerStatusMonitor`

> A terminal UI for monitoring servers and services — checks HTTP/HTTPS endpoints, raw TCP ports, and GitHub Actions workflows. Tracks uptime history and sounds an alarm when something goes down.

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)
![SQLite](https://img.shields.io/badge/SQLite-uptime_log-003B57?style=flat-square&logo=sqlite)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)

</div>

---

## 📸 Demo

```
╭─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│  Name                          URL                                                  Status        Code    Latency   Uptime  │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│> HTTPS Check NET               https://www.cloudflare.com                           ●  UP          200     43ms      99.98% │
│  TCP Check GOOG                tcp://8.8.8.8:53                                     ●  UP          —       4ms       100.00%│
│  Test 500                      https://httpstat.us/500                              ⬢  DOWN        500     —         12.50% │
│  Test 404                      https://httpstat.us/404                              ◆  WARN        404     —         87.20% │
│  └ connection refused: dial tcp 0.0.0.0:404                                                                                 │
│  Test CI Failure               gh-actions://sindresorhus/got/refs/heads/main        ⬢  DOWN        —       212ms     80.00% │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│  ↑/↓ or ←/→ navigate · r recheck · R recheck all · a mute · q quit · next check in 7s                                      │
╰─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
```

---

## ✨ Features

- 🌐 **HTTP/HTTPS monitoring** — checks any endpoint, treats redirects as their own state
- 🔌 **TCP monitoring** — raw socket dial for non-HTTP services (DNS, databases, etc.)
- 🔁 **GitHub Actions monitoring** — polls the latest workflow run on any branch via `gh-actions://`
- 🚦 **Four-state status display** — `UP`, `RDR` (redirect), `WARN` (4xx / in-progress CI), `DOWN` (5xx / failed CI)
- 📊 **24h uptime percentage** — calculated from a local SQLite event log
- 🔔 **Audio alarm** — plays `alarm.wav` on loop when any server is down; toggle mute with `a`
- ⏱️ **Per-server intervals & timeouts** — override the global poll interval per entry
- ⌨️ **Keyboard-driven** — no mouse required
- 💾 **Persistent history** — every check result is written to `uptime.db` (SQLite)

---

## 📦 Installation

### From Source

```bash
git clone https://github.com/chasehaye/ServerStatusMonitor.git
cd ServerStatusMonitor
go build -o ServerStatusMonitor .
```

### Requirements

- [Go 1.21+](https://go.dev/dl/)
- `gcc` — required by the `go-sqlite3` driver (CGO)
- [`ffplay`](https://ffmpeg.org/) — required for the audio alarm (`ffmpeg` package includes it)

**macOS:**
```bash
brew install ffmpeg
```

**Linux (Debian/Ubuntu):**
```bash
sudo apt install ffmpeg gcc
```

**Arch Linux:**
```bash
sudo pacman -S ffmpeg gcc
```

---

## 🚀 Usage

```bash
# Run with config.yaml in the current directory
./ServerStatusMonitor
```

On startup the app:
1. Reads `config.yaml` from the current directory
2. Opens (or creates) `uptime.db` for event logging
3. Immediately checks all servers, then polls on each server's configured interval

Place an `alarm.wav` file in the same directory as the binary if you want audio alerts.

For GitHub Actions monitoring on private repos, set a personal access token:
```bash
export GITHUB_TOKEN=your_token_here
./ServerStatusMonitor
```

Public repos work without a token but will be rate-limited by GitHub's API (60 req/hr unauthenticated).

---

## ⌨️ Keybindings

| Key | Action |
|-----|--------|
| `↑` / `←` | Move selection up |
| `↓` / `→` | Move selection down |
| `r` | Recheck the selected server |
| `R` | Recheck all servers |
| `a` | Toggle alarm mute |
| `q` / `Ctrl+C` | Quit |

---

## ⚙️ Configuration

The app reads `config.yaml` from the working directory. Place it alongside the binary or run from the directory that contains it.

### Example `config.yaml`

```yaml
# Global poll interval (used when a server doesn't define its own)
interval: 60s

servers:
  - name: HTTPS Check NET
    url: https://www.cloudflare.com
    interval: 50s
    timeout: 3s

  - name: TCP Check GOOG
    url: tcp://8.8.8.8:53
    interval: 11s
    timeout: 8s

  - name: Test 500
    url: https://httpstat.us/500
    interval: 5s
    timeout: 35s

  - name: Test 404
    url: https://httpstat.us/404
    interval: 5s
    timeout: 3s

  - name: Test CI Failure
    url: gh-actions://sindresorhus/got/refs/heads/repro-2444
    interval: 120s
    timeout: 10s
```

### Configuration Reference

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `interval` | duration | `10s` | Global poll interval applied to all servers that don't set their own |
| `servers[].name` | string | — | Display name shown in the TUI |
| `servers[].url` | string | — | URL to check — see formats below |
| `servers[].interval` | duration | _(global)_ | Per-server poll interval; omit to use the global value |
| `servers[].timeout` | duration | `5s` | Per-server connection/response timeout |

### URL Formats

| Format | Example | Notes |
|--------|---------|-------|
| HTTP/HTTPS | `https://example.com` | 3xx treated as `RDR`, 4xx as `WARN`, 5xx as `DOWN` |
| TCP | `tcp://8.8.8.8:53` | Raw socket dial — no HTTP involved |
| GitHub Actions | `gh-actions://owner/repo` | Checks latest run on default branch |
| GitHub Actions (branch) | `gh-actions://owner/repo/refs/heads/main` | Checks latest run on a specific branch |

### GitHub Actions Status Mapping

| Run state | TUI status |
|-----------|-----------|
| `success` | `●  UP` |
| `in_progress` or `queued` | `»  RDR` (up with warning) |
| `failure` | `⬢  DOWN` |
| anything else / no runs | `◆  WARN` |

---

## 📊 Status Indicators

| Symbol | Label | Meaning |
|--------|-------|---------|
| `●` | UP | 2xx / TCP connect success / CI passed |
| `»` | RDR | 3xx redirect or CI run in progress |
| `◆` | WARN | 4xx client error or no CI runs found |
| `⬢` | DOWN | 5xx server error, connection failure, or CI failed |

HTTP status codes are colour-coded in the **Code** column: green (2xx), yellow (3xx), orange (4xx), red (5xx).

---

## 🗄️ Uptime Database

All check results are appended to `uptime.db` (SQLite) in the working directory. The schema is:

```sql
CREATE TABLE events (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    name  TEXT    NOT NULL,
    url   TEXT    NOT NULL,
    up    BOOLEAN NOT NULL,
    ts    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

The **Uptime** column shows the percentage of successful checks in the **last 24 hours**, recalculated after every result.

---

## 🛠️ Development

### Project Structure

```
ServerStatusMonitor/
├── main.go          # Bubble Tea model, Init/Update/View
├── check.go         # HTTP, TCP & GitHub Actions check logic
├── config.go        # Config & Server types, YAML loading
├── db.go            # SQLite init, event logging, uptime calculation
├── config.yaml      # Your server list (add to .gitignore)
├── alarm.wav        # Audio file played when a server is down
└── uptime.db        # Auto-created SQLite database
```

### Dependencies

| Package | Purpose |
|---------|---------|
| [`charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea) | TUI framework |
| [`charmbracelet/bubbles`](https://github.com/charmbracelet/bubbles) | Spinner component |
| [`charmbracelet/lipgloss`](https://github.com/charmbracelet/lipgloss) | Terminal styling |
| [`mattn/go-sqlite3`](https://github.com/mattn/go-sqlite3) | SQLite driver (CGO) |
| [`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3) | YAML config parsing |

### Build & Run

```bash
go mod tidy
go run .

# or build a binary
go build -o ServerStatusMonitor .
./ServerStatusMonitor
```

---

## 🤝 Contributing

Contributions are welcome! Here's how:

1. **Fork** the repo and clone it locally
2. **Create a branch** for your change:
   ```bash
   git checkout -b feat/my-feature
   ```
3. **Make your changes** — keep PRs focused on one thing
4. **Test manually** — run the app against real/mock endpoints
5. **Commit** with a clear message:
   ```bash
   git commit -m "feat: add latency threshold warnings"
   ```
6. **Open a Pull Request** and describe what you changed and why

### Good First Ideas

- Add a `--config` flag to specify a custom config path
- Support `j`/`k` vim keybindings
- Add a detail pane showing recent event history for the selected server
- Export uptime stats to JSON or CSV
- Support more CI providers (GitLab, Bitbucket Pipelines)

### Reporting Bugs

Open an [issue](https://github.com/chasehaye/ServerStatusMonitor/issues) and include:
- Your OS, Go version, and `ffmpeg` version
- Your `config.yaml` (redact any sensitive hostnames/IPs)
- What you expected vs what happened

---

## 📄 License

MIT © [Chase Haye](https://github.com/chasehaye)