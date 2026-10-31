// Command tgexport converts Telegram Desktop JSON chat exports to plain
// text, HTML, Markdown, CSV or simplified JSON.
package main

import (
	"os"
	"runtime/debug"

	"github.com/antalkon/tgexport/internal/cli"
)

// Set by GoReleaser via -ldflags "-X main.version=... -X main.commit=... -X main.date=...".
var (
	version = ""
	commit  = ""
	date    = ""
)

func main() {
	app := &cli.App{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Build:  buildInfo(),
	}
	os.Exit(app.Run(os.Args[1:]))
}

// buildInfo falls back to module data for "go install ...@vX" builds, which
// do not pass ldflags.
func buildInfo() cli.BuildInfo {
	b := cli.BuildInfo{Version: version, Commit: commit, Date: date}
	if b.Version != "" {
		return b
	}
	b.Version = "dev"
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			b.Version = v
		}
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				if len(s.Value) > 7 {
					b.Commit = s.Value[:7]
				}
			case "vcs.time":
				b.Date = s.Value
			}
		}
	}
	return b
}
