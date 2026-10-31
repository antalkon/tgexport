package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture(name string) string {
	p, _ := filepath.Abs(filepath.Join("../../testdata", name))
	return p
}

type result struct {
	code           int
	stdout, stderr string
}

// run executes the CLI with dir as the working directory for default output.
func run(t *testing.T, dir string, args ...string) result {
	t.Helper()
	var out, errb bytes.Buffer
	app := &App{
		Stdout: &out, Stderr: &errb,
		Build: BuildInfo{Version: "v1.2.3", Commit: "abc1234", Date: "2026-10-31"},
		Getwd: func() (string, error) { return dir, nil },
	}
	code := app.Run(args)
	return result{code, out.String(), errb.String()}
}

func TestDefaultTxtToWorkingDir(t *testing.T) {
	dir := t.TempDir()
	r := run(t, dir, fixture("personal.json"))
	if r.code != ExitOK {
		t.Fatalf("exit %d: %s", r.code, r.stderr)
	}
	data, err := os.ReadFile(filepath.Join(dir, "Alice.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "2026-10-01 09:00:00 Bob: Morning!\n") {
		t.Errorf("unexpected txt:\n%s", data)
	}
	if !strings.Contains(r.stderr, "6 messages") {
		t.Errorf("summary missing: %q", r.stderr)
	}
}

func TestHTMLFlagAfterInput(t *testing.T) {
	dir := t.TempDir()
	r := run(t, dir, fixture("group.json"), "--html", "-q")
	if r.code != ExitOK {
		t.Fatalf("exit %d: %s", r.code, r.stderr)
	}
	// "/" in the chat title must not create directories.
	if _, err := os.Stat(filepath.Join(dir, "Project_ team _ chat.html")); err != nil {
		t.Fatal(err)
	}
	if r.stderr != "" {
		t.Errorf("--quiet still printed: %q", r.stderr)
	}
}

func TestStdout(t *testing.T) {
	r := run(t, t.TempDir(), "-o", "-", "--format", "csv", fixture("personal.json"))
	if r.code != ExitOK {
		t.Fatalf("exit %d: %s", r.code, r.stderr)
	}
	if !strings.HasPrefix(r.stdout, "id,date,time,from") {
		t.Errorf("stdout = %q", r.stdout)
	}
}

func TestOutputFileAndDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "nested", "out.md")
	if r := run(t, dir, fixture("personal.json"), "-f", "md", "-o", file); r.code != ExitOK {
		t.Fatalf("exit %d: %s", r.code, r.stderr)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(dir, "exports") + string(filepath.Separator)
	if r := run(t, dir, fixture("account.json"), "-o", outDir); r.code != ExitOK {
		t.Fatalf("exit %d: %s", r.code, r.stderr)
	}
	entries, _ := os.ReadDir(outDir)
	if len(entries) != 3 {
		t.Fatalf("got %d files, want 3 (two chats named Alice must not collide)", len(entries))
	}
}

func TestAccountExportSelectChat(t *testing.T) {
	r := run(t, t.TempDir(), fixture("account.json"), "--chat", "5005", "-o", "-")
	if r.code != ExitOK {
		t.Fatalf("exit %d: %s", r.code, r.stderr)
	}
	if !strings.Contains(r.stdout, "Dave") || strings.Contains(r.stdout, "Morning") {
		t.Errorf("wrong chat selected:\n%s", r.stdout)
	}
	if r := run(t, t.TempDir(), fixture("account.json"), "--chat", "nope"); r.code != ExitError {
		t.Errorf("unknown chat: exit %d", r.code)
	}
}

func TestUsageErrors(t *testing.T) {
	tests := [][]string{
		{},
		{"a.json", "b.json"},
		{"--format", "pdf", fixture("personal.json")},
		{"--from", "yesterday", fixture("personal.json")},
		{"--from", "2026-10-05", "--to", "2026-10-01", fixture("personal.json")},
		{"--bogus", fixture("personal.json")},
		{fixture("account.json"), "-o", "-"},
	}
	for _, args := range tests {
		if r := run(t, t.TempDir(), args...); r.code != ExitUsage {
			t.Errorf("%v: exit %d, want %d (stderr %q)", args, r.code, ExitUsage, r.stderr)
		}
	}
}

func TestMissingFile(t *testing.T) {
	r := run(t, t.TempDir(), "does-not-exist.json")
	if r.code != ExitError || !strings.Contains(r.stderr, "does-not-exist.json") {
		t.Errorf("exit %d, stderr %q", r.code, r.stderr)
	}
}

func TestVersionAndHelp(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"--version"}, {"-v"}} {
		r := run(t, "", args...)
		if r.code != ExitOK || !strings.Contains(r.stdout, "v1.2.3") {
			t.Errorf("%v: exit %d, stdout %q", args, r.code, r.stdout)
		}
	}
	r := run(t, "", "--help")
	if r.code != ExitOK || !strings.Contains(r.stderr, "Usage:") {
		t.Errorf("help: exit %d, stderr %q", r.code, r.stderr)
	}
}

func TestStats(t *testing.T) {
	r := run(t, "", "stats", fixture("group.json"))
	if r.code != ExitOK {
		t.Fatalf("exit %d: %s", r.code, r.stderr)
	}
	for _, want := range []string{"Project: team / chat", "Messages:", "6 (text 3, media 1, service 2)", "Dave"} {
		if !strings.Contains(r.stdout, want) {
			t.Errorf("stats missing %q:\n%s", want, r.stdout)
		}
	}
}

func TestSafeFileName(t *testing.T) {
	tests := map[string]string{
		"Alice":               "Alice",
		`a/b\c:d*e?f"g<h>i|j`: "a_b_c_d_e_f_g_h_i_j",
		"  ..dots..  ":        "dots",
		"":                    "chat",
		"Привет, мир 👋":       "Привет, мир 👋",
	}
	for in, want := range tests {
		if got := SafeFileName(in); got != want {
			t.Errorf("SafeFileName(%q) = %q, want %q", in, got, want)
		}
	}
	if n := len([]rune(SafeFileName(strings.Repeat("я", 300)))); n != 100 {
		t.Errorf("long name truncated to %d runes, want 100", n)
	}
}

func TestParseDate(t *testing.T) {
	to, err := parseDate("2026-10-01", true)
	if err != nil {
		t.Fatal(err)
	}
	if got := to.Format("15:04:05"); got != "23:59:59" {
		t.Errorf("end of day = %s", got)
	}
	from, err := parseDate("2026-10-01 12:30", false)
	if err != nil {
		t.Fatal(err)
	}
	if !from.Equal(time.Date(2026, 10, 1, 12, 30, 0, 0, time.Local)) {
		t.Errorf("from = %v", from)
	}
}
