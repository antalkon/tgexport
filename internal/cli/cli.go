// Package cli implements the tgexport command line.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/antalkon/tgexport/internal/export"
	"github.com/antalkon/tgexport/internal/telegram"
)

// Exit codes.
const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2
)

// BuildInfo is injected by main via -ldflags at release time.
type BuildInfo struct {
	Version, Commit, Date string
}

// App holds the I/O the CLI talks to, so tests can run it in-process.
type App struct {
	Stdout, Stderr io.Writer
	Build          BuildInfo
	// Getwd returns the default output directory; os.Getwd when nil.
	Getwd func() (string, error)
}

// errUsage marks errors caused by bad invocation (exit code 2).
var errUsage = errors.New("usage")

type usageError struct{ msg string }

func (e usageError) Error() string   { return e.msg }
func (e usageError) Is(t error) bool { return t == errUsage }

func usagef(format string, a ...any) error { return usageError{fmt.Sprintf(format, a...)} }

// Run executes the CLI with args (without the program name) and returns the
// process exit code.
func (a *App) Run(args []string) int {
	err := a.run(args)
	switch {
	case err == nil:
		return ExitOK
	case errors.Is(err, flag.ErrHelp):
		return ExitOK
	case errors.Is(err, errUsage):
		fmt.Fprintf(a.Stderr, "tgexport: %v\nRun 'tgexport --help' for usage.\n", err)
		return ExitUsage
	default:
		fmt.Fprintf(a.Stderr, "tgexport: %v\n", err)
		return ExitError
	}
}

func (a *App) run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "version":
			a.printVersion()
			return nil
		case "stats":
			return a.runStats(args[1:])
		case "help":
			args = []string{"--help"}
		}
	}
	return a.runExport(args)
}

func (a *App) printVersion() {
	fmt.Fprintf(a.Stdout, "tgexport %s (commit %s, built %s)\n",
		orDash(a.Build.Version), orDash(a.Build.Commit), orDash(a.Build.Date))
}

type exportFlags struct {
	format    string
	html      bool
	output    string
	chat      string
	from, to  string
	users     multiFlag
	noService bool
	noMedia   bool
	me        string
	layout    string
	dateFmt   string
	timeFmt   string
	quiet     bool
	version   bool
}

func (a *App) newExportFlagSet(f *exportFlags) *flag.FlagSet {
	fs := flag.NewFlagSet("tgexport", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)

	fs.StringVar(&f.format, "format", "txt", "")
	fs.StringVar(&f.format, "f", "txt", "")
	fs.BoolVar(&f.html, "html", false, "")
	fs.StringVar(&f.output, "output", "", "")
	fs.StringVar(&f.output, "o", "", "")
	fs.StringVar(&f.chat, "chat", "", "")
	fs.StringVar(&f.from, "from", "", "")
	fs.StringVar(&f.to, "to", "", "")
	fs.Var(&f.users, "user", "")
	fs.Var(&f.users, "u", "")
	fs.BoolVar(&f.noService, "no-service", false, "")
	fs.BoolVar(&f.noMedia, "no-media", false, "")
	fs.StringVar(&f.me, "me", "", "")
	fs.StringVar(&f.layout, "layout", export.DefaultLayout, "")
	fs.StringVar(&f.dateFmt, "date-format", export.DefaultDateFormat, "")
	fs.StringVar(&f.timeFmt, "time-format", export.DefaultTimeFormat, "")
	fs.BoolVar(&f.quiet, "quiet", false, "")
	fs.BoolVar(&f.quiet, "q", false, "")
	fs.BoolVar(&f.version, "version", false, "")
	fs.BoolVar(&f.version, "v", false, "")
	fs.Usage = func() { fmt.Fprint(a.Stderr, usageText) }
	return fs
}

func (a *App) runExport(args []string) error {
	var f exportFlags
	fs := a.newExportFlagSet(&f)
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return usageError{err.Error()}
	}
	if f.version {
		a.printVersion()
		return nil
	}
	if len(pos) != 1 {
		fs.Usage()
		return usagef("expected exactly one input file, got %d", len(pos))
	}
	if f.html {
		f.format = "html"
	}
	format, err := export.LookupFormat(f.format)
	if err != nil {
		return usageError{err.Error()}
	}
	opts, err := f.options()
	if err != nil {
		return err
	}

	chats, err := telegram.Load(pos[0])
	if err != nil {
		return err
	}
	chats, err = selectChats(chats, f.chat)
	if err != nil {
		return err
	}

	targets, err := a.resolveTargets(f.output, chats, format.Ext())
	if err != nil {
		return err
	}
	for i := range chats {
		doc := export.NewDocument(&chats[i], opts)
		if err := a.write(targets[i], format, doc); err != nil {
			return err
		}
		if !f.quiet && targets[i] != "-" {
			fmt.Fprintf(a.Stderr, "✓ %s → %s (%d messages)\n", chats[i].Title(), targets[i], len(doc.Messages))
		}
	}
	return nil
}

func (f *exportFlags) options() (export.Options, error) {
	opts := export.Options{
		Users:      f.users,
		NoService:  f.noService,
		NoMedia:    f.noMedia,
		Me:         f.me,
		Layout:     f.layout,
		DateFormat: f.dateFmt,
		TimeFormat: f.timeFmt,
	}
	var err error
	if opts.Since, err = parseDate(f.from, false); err != nil {
		return opts, usagef("--from: %v", err)
	}
	if opts.Until, err = parseDate(f.to, true); err != nil {
		return opts, usagef("--to: %v", err)
	}
	if !opts.Since.IsZero() && !opts.Until.IsZero() && opts.Until.Before(opts.Since) {
		return opts, usagef("--to is before --from")
	}
	return opts, nil
}

// parseDate accepts "2006-01-02", "2006-01-02 15:04" or "2006-01-02T15:04:05".
// A bare date used as an upper bound covers the whole day.
func parseDate(s string, endOfDay bool) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		if endOfDay {
			t = t.Add(24*time.Hour - time.Nanosecond)
		}
		return t, nil
	}
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02T15:04", "2006-01-02 15:04:05", telegram.TimeLayout} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date %q (want YYYY-MM-DD or \"YYYY-MM-DD HH:MM\")", s)
}

// selectChats narrows a multi-chat (full account) export down to the chat
// requested with --chat, matched by exact ID or case-insensitive name.
func selectChats(chats []telegram.Chat, query string) ([]telegram.Chat, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return chats, nil
	}
	var out []telegram.Chat
	for _, c := range chats {
		if fmt.Sprint(c.ID) == query || strings.EqualFold(c.Name, query) {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no chat matches %q; run 'tgexport stats <file>' to list chats", query)
	}
	return out, nil
}

// resolveTargets maps each chat to an output path ("-" means stdout).
func (a *App) resolveTargets(output string, chats []telegram.Chat, ext string) ([]string, error) {
	multi := len(chats) > 1
	if output == "-" {
		if multi {
			return nil, usagef("cannot write %d chats to stdout; use --chat or an output directory", len(chats))
		}
		return []string{"-"}, nil
	}

	dir := output
	if output != "" && !isDirTarget(output) {
		if multi {
			return nil, usagef("%d chats found: --output must be a directory (or pick one with --chat)", len(chats))
		}
		return []string{output}, nil
	}
	if dir == "" {
		getwd := a.Getwd
		if getwd == nil {
			getwd = os.Getwd
		}
		var err error
		if dir, err = getwd(); err != nil {
			return nil, err
		}
	}

	seen := map[string]bool{}
	out := make([]string, len(chats))
	for i, c := range chats {
		name := SafeFileName(c.Title())
		if seen[name] {
			name = fmt.Sprintf("%s_%d", name, c.ID)
		}
		seen[name] = true
		out[i] = filepath.Join(dir, name+"."+ext)
	}
	return out, nil
}

func isDirTarget(p string) bool {
	if strings.HasSuffix(p, "/") || strings.HasSuffix(p, string(filepath.Separator)) {
		return true
	}
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func (a *App) write(target string, f export.Format, doc *export.Document) (err error) {
	if target == "-" {
		return f.Render(a.Stdout, doc)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	// Write to a temp file and rename, so a failed export never leaves a
	// truncated file behind.
	tmp, err := os.CreateTemp(filepath.Dir(target), ".tgexport-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	if err = f.Render(tmp, doc); err != nil {
		return fmt.Errorf("render %s: %w", target, err)
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), target)
}

// SafeFileName turns a chat title into a portable file name.
func SafeFileName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r < 32, strings.ContainsRune(`<>:"/\|?*`, r):
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	name := strings.Trim(strings.TrimSpace(b.String()), ". ")
	if r := []rune(name); len(r) > 100 {
		name = strings.TrimSpace(string(r[:100]))
	}
	if name == "" {
		name = "chat"
	}
	return name
}

// parseInterspersed lets flags appear before and after positional arguments
// ("tgexport chat.json --html"), which the standard flag package doesn't.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			return pos, nil
		}
		if args[0] == "--" {
			return append(pos, args[1:]...), nil
		}
		pos = append(pos, args[0])
		args = args[1:]
	}
}

// multiFlag collects repeated and comma-separated values.
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }

func (m *multiFlag) Set(v string) error {
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			*m = append(*m, p)
		}
	}
	return nil
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
