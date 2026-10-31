# tgexport

[![CI](https://github.com/antalkon/tgexport/actions/workflows/ci.yml/badge.svg)](https://github.com/antalkon/tgexport/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/antalkon/tgexport)](https://github.com/antalkon/tgexport/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/antalkon/tgexport.svg)](https://pkg.go.dev/github.com/antalkon/tgexport)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**English** · [Русский](README.ru.md)

`tgexport` turns a Telegram Desktop chat export (`result.json`) into something
you can actually read, search or feed into other tools: plain text, a
self-contained HTML page, Markdown, CSV or simplified JSON.

```text
$ tgexport result.json
✓ Alice → /Users/me/Alice.txt (163 messages)

$ head -3 Alice.txt
2026-10-22 22:09:09 Alice: what does that mean
2026-10-22 22:20:51 Bob: [GIF]
2026-10-22 22:28:49 Bob: do you like water?
```

- Works with **personal chats, groups, channels** and **full account exports**.
- **Single static binary**, no dependencies, macOS / Linux / Windows, amd64 / arm64.
- Formats: **txt** (default), **html**, **md**, **csv**, **json**.
- Filters by **date range**, **author**, drops **service** messages or **media** placeholders.
- HTML output is one offline file: dark/light theme, replies, reactions, forwarded
  messages, formatting, code blocks and a built-in search box.
- Media is shown as readable placeholders: `[photo]`, `[sticker 👍]`, `[voice message 0:02]`…

## Contents

- [Install](#install)
- [Update](#update)
- [Uninstall](#uninstall)
- [Getting a JSON export from Telegram](#getting-a-json-export-from-telegram)
- [Usage](#usage)
- [Output formats](#output-formats)
- [Flags](#flags)
- [Examples](#examples)
- [FAQ](#faq)
- [Development](#development)

## Install

**macOS / Linux**

```sh
curl -fsSL https://raw.githubusercontent.com/antalkon/tgexport/main/scripts/install.sh | sh
```

The binary goes to `~/.local/bin` (or wherever an existing `tgexport` already
lives). If that directory is not on your `PATH`, the installer adds it to your
shell profile — restart the terminal afterwards. Options:

```sh
# a specific version, a custom directory, without touching shell profiles
curl -fsSL https://raw.githubusercontent.com/antalkon/tgexport/main/scripts/install.sh \
  | sh -s -- --version v0.1.0 --dir /usr/local/bin --no-modify-path
```

**Windows (PowerShell)**

```powershell
irm https://raw.githubusercontent.com/antalkon/tgexport/main/scripts/install.ps1 | iex
```

Installs to `%LOCALAPPDATA%\Programs\tgexport` and adds it to your user `PATH`
(open a new terminal afterwards). No administrator rights needed.

**With Go** (1.23+)

```sh
go install github.com/antalkon/tgexport/cmd/tgexport@latest
```

**Manually** — download the archive for your platform from
[Releases](https://github.com/antalkon/tgexport/releases/latest), check it
against `checksums.txt`, unpack and put `tgexport` somewhere on your `PATH`.

Both install scripts verify the SHA-256 checksum of the downloaded archive
before installing.

## Update

Run the same install command again. The script compares the installed version
with the latest release and either updates the binary in place or prints
`already installed — nothing to do`. Add `--force` (`-Force` on Windows) to
reinstall anyway.

Check which version you have with `tgexport version`.

## Uninstall

**macOS / Linux**

```sh
curl -fsSL https://raw.githubusercontent.com/antalkon/tgexport/main/scripts/uninstall.sh | sh
```

Removes the `tgexport` binaries found on your `PATH` and in the default
locations, and the `PATH` line the installer added to your shell profile.

**Windows (PowerShell)**

```powershell
irm https://raw.githubusercontent.com/antalkon/tgexport/main/scripts/uninstall.ps1 | iex
```

**Go**: `rm "$(go env GOPATH)/bin/tgexport"`.

## Getting a JSON export from Telegram

1. Open **Telegram Desktop** (the export is not available in mobile apps).
2. Open the chat → **⋮** menu → **Export chat history**.
3. Untick media you don't need, set **Format: Machine-readable JSON**, click **Export**.
4. You get a folder like `ChatExport_2026-10-31/` with `result.json` inside.

A whole-account export (**Settings → Advanced → Export Telegram data**, JSON
format) works too; see [`--chat`](#flags).

## Usage

```text
tgexport [flags] <result.json | export-folder>
tgexport stats <result.json | export-folder>
tgexport version
```

- The input is a path to `result.json` **or** to the export folder containing it.
- By default the result is written to **the current directory** as
  `<chat name>.<ext>`, e.g. `Alice.txt`. An existing file with that name is
  overwritten. Characters that are not allowed in file names (`/ \ : * ? " < > |`)
  are replaced with `_`.
- Flags may go before or after the input file.
- Exit codes: `0` success, `1` runtime error (missing file, bad JSON…),
  `2` invalid usage.

`tgexport stats` prints a quick summary:

```text
Chat:         Project chat (private_group, id 5549517217)
Period:       2026-09-21 — 2026-10-24
Messages:     372 (text 317, media 55, service 6)
Busiest day:  2026-09-25 (146 messages)
Authors:
    147  40.2%  Alice
    112  30.6%  Bob
```

## Output formats

| Format | Flag | Description |
| --- | --- | --- |
| `txt` | *(default)* | One message per line: `date time user: text`. Multi-line messages continue on lines indented by 4 spaces, so every message starts at column 0 and `grep` works. |
| `html` | `--html` or `-f html` | A single offline page that looks like a chat: day separators, reply quotes (click to jump), reactions, edited marks, forwarded labels, links, code blocks, spoilers, search box, dark mode. |
| `md` | `-f md` | Markdown with a section per day; keeps bold/italic/code/links. |
| `csv` | `-f csv` | `id,date,time,from,from_id,type,reply_to,forwarded_from,media,text,reactions` — for spreadsheets and pandas. |
| `json` | `-f json` | A flat, stable JSON structure (`id`, `date`, `from`, `text`, `media`, `reply_to`, `reactions`…) that is much easier to process than Telegram's raw format. |

What appears in the text for non-text content:

| In Telegram | In tgexport |
| --- | --- |
| photo | `[photo]` |
| sticker | `[sticker 🤔]` |
| voice / video message | `[voice message 0:02]`, `[video message 0:03]` |
| GIF | `[GIF]` |
| video, audio, file | `[video clip.mp4 0:15]`, `[audio song.mp3 3:20]`, `[file report.pdf]` |
| poll, location, contact | `[poll Lunch?]`, `[location 57.43000, 56.94000]`, `[contact Ann +1…]` |
| forwarded message | `[forwarded from Channel] text` |
| service message | `Alice: * pinned a message`, `* added Bob, Carol`, `* joined the group via invite link` |
| text link | `docs (https://example.com/docs)` |

## Flags

| Flag | Default | Description |
| --- | --- | --- |
| `-f, --format` | `txt` | `txt`, `html`, `md`, `csv`, `json` |
| `--html` | | Shortcut for `--format html` |
| `-o, --output` | current dir | Output file, directory (existing, or ending with `/`), or `-` for stdout |
| `--from` | | Keep messages on or after this date: `YYYY-MM-DD` or `"YYYY-MM-DD HH:MM"` |
| `--to` | | Keep messages on or before this date (a bare date includes the whole day) |
| `-u, --user` | | Keep messages by this author name or ID (`user123`); repeatable or comma-separated; case-insensitive |
| `--no-service` | | Drop service messages (joins, pins, calls…) |
| `--no-media` | | Drop media placeholders; messages that become empty are skipped |
| `--chat` | | For full account exports: export only the chat with this name or ID |
| `--layout` | `{date} {time} {user}: {text}` | Line template for `txt`. Placeholders: `{date}` `{time}` `{user}` `{user_id}` `{id}` `{text}` |
| `--date-format` | `2006-01-02` | [Go time layout](https://pkg.go.dev/time#pkg-constants) for `{date}` |
| `--time-format` | `15:04:05` | Go time layout for `{time}` |
| `--me` | | HTML: draw this author's messages on the right, like your own messages in Telegram |
| `-q, --quiet` | | Don't print the `✓ chat → file` summary |
| `-v, --version` | | Print version |
| `-h, --help` | | Print help |

## Examples

```sh
# export folder → Alice.html in the current directory
tgexport ~/Downloads/"Telegram Desktop"/ChatExport_2026-10-31 --html

# your messages on the right
tgexport result.json --html --me "Alice"

# October 1–15 only, only Bob and Carol, no joins/pins
tgexport result.json --from 2026-10-01 --to 2026-10-15 -u Bob,Carol --no-service

# colon-separated layout: 2026-10-22:22:09:09:Alice what does that mean
tgexport result.json --layout "{date}:{time}:{user} {text}"

# Russian-style dates: 22.10.2026 22:09 Alice: …
tgexport result.json --date-format 02.01.2006 --time-format 15:04

# to stdout, pipe into other tools
tgexport result.json -o - | grep -i deadline
tgexport result.json -f csv -o - | wc -l

# full account export: every chat into ./exports/, or just one
tgexport ~/Downloads/DataExport_2026-10-31 -f md -o exports/
tgexport ~/Downloads/DataExport_2026-10-31 --chat "Alice"
```

## FAQ

**Are photos and files copied into the HTML?** No. tgexport works with
`result.json` only and shows attachments as placeholders. The HTML file is
fully self-contained and makes no network requests.

**Times are off by a few hours.** Telegram writes timestamps in the time zone
of the computer that made the export, without a zone marker. tgexport prints
them as they are.

**The file name looks strange.** It comes from the chat name. Use `-o name.txt`
to choose your own.

**Windows says the command isn't found right after installing.** Open a new
terminal — `PATH` changes apply only to new sessions.

**Is my data sent anywhere?** No. tgexport is fully offline.

## Development

```sh
git clone https://github.com/antalkon/tgexport && cd tgexport
make test       # go test -race ./...
make lint       # gofmt, go vet, golangci-lint
make build      # ./bin/tgexport
make golden     # regenerate golden files after an intended output change
make snapshot   # cross-compile all platforms with GoReleaser into ./dist
```

Project layout:

```text
cmd/tgexport/        entry point (version info via ldflags)
internal/telegram/   export JSON model, parsing, media/service descriptions
internal/export/     filtering + renderers (txt, html, md, csv, json)
internal/cli/        flags, output paths, stats subcommand
scripts/             install / update / uninstall scripts
testdata/            synthetic sample exports used by tests
```

Adding a format means implementing `export.Format` (`Name`, `Ext`, `Render`)
and registering it in `init()`; see [CONTRIBUTING.md](CONTRIBUTING.md).

Releases are cut by pushing a `vX.Y.Z` tag: GitHub Actions runs tests,
GoReleaser builds archives for all platforms with checksums, and a smoke test
installs, re-runs and uninstalls the release on macOS, Linux and Windows.

## License

[MIT](LICENSE)
