# Contributing

Thanks for your interest in tgexport! Issues and pull requests are welcome.

## Setup

You need Go 1.23+ and, optionally, [golangci-lint](https://golangci-lint.run)
and [GoReleaser](https://goreleaser.com).

```sh
git clone https://github.com/antalkon/tgexport && cd tgexport
make test lint
```

## Guidelines

- Keep the binary dependency-free: the standard library is enough so far.
- Run `make lint test` before pushing; CI runs the same checks on Linux,
  macOS and Windows.
- Output formats are covered by golden files in `internal/export/testdata`.
  After an intentional output change run `make golden` and review the diff.
- **Never commit real chat exports.** Test fixtures in `testdata/` are
  synthetic; extend them when you need new message shapes.
- Use [Conventional Commits](https://www.conventionalcommits.org) (`feat:`,
  `fix:`, `docs:`…); release notes are generated from them.

## Adding an output format

1. Create `internal/export/<name>.go` with a type implementing `export.Format`:

   ```go
   type xmlFormat struct{}

   func (xmlFormat) Name() string { return "xml" }
   func (xmlFormat) Ext() string  { return "xml" }
   func (xmlFormat) Render(w io.Writer, d *Document) error { /* ... */ }

   func init() { register(xmlFormat{}) }
   ```

2. Add it to the golden test list in `export_test.go` and run `make golden`.
3. Document it in both READMEs.

## Releasing

1. Move the `Unreleased` entries in `CHANGELOG.md` under a new version heading.
2. Tag and push:

   ```sh
   git tag -a v0.2.0 -m "v0.2.0"
   git push origin v0.2.0
   ```

3. The `Release` workflow tests, builds and publishes the archives, then
   smoke-tests the install scripts on every OS.

## Reporting issues

Please include `tgexport version`, your OS, the command you ran and — if the
problem is parsing — a **minimal, anonymised** snippet of `result.json`.
