# Changelog

All notable changes to this project are documented here. The format is based
on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project
follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0] - 2026-10-31

### Added

- Convert Telegram Desktop JSON exports (`result.json` or the export folder) to
  `txt` (default), `html`, `md`, `csv` and `json`.
- Personal chats, groups, channels and full account exports (`--chat` to pick one).
- Filters: `--from`, `--to`, `--user`, `--no-service`, `--no-media`.
- Text layout customization: `--layout`, `--date-format`, `--time-format`.
- Self-contained HTML with replies, reactions, forwarded labels, formatting,
  search and dark mode; `--me` to align your own messages to the right.
- `tgexport stats` subcommand.
- Install / update / uninstall scripts for macOS, Linux and Windows with
  SHA-256 verification.

[Unreleased]: https://github.com/antalkon/tgexport/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/antalkon/tgexport/releases/tag/v0.1.0
