// Package export filters chat messages and renders them into output formats.
package export

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/antalkon/tgexport/internal/telegram"
)

// Options controls which messages are exported and how they look.
type Options struct {
	// Since and Until bound message dates (inclusive). Zero means unbounded.
	Since, Until time.Time
	// Users keeps only messages whose author name or ID matches one of the
	// entries (case-insensitive). Empty keeps everyone.
	Users []string
	// NoService drops service messages (joins, pins, calls...).
	NoService bool
	// NoMedia drops media placeholders; messages left empty are skipped.
	NoMedia bool
	// Me is the author shown on the right side in HTML output.
	Me string
	// Layout is the line template for the text format.
	Layout string
	// DateFormat and TimeFormat are Go time layouts for the text format.
	DateFormat, TimeFormat string
}

// Document is a filtered chat ready for rendering.
type Document struct {
	Chat     *telegram.Chat
	Messages []telegram.Message
	Opts     Options

	byID map[int64]*telegram.Message
}

// NewDocument applies opts to chat and indexes messages for reply lookups.
func NewDocument(chat *telegram.Chat, opts Options) *Document {
	d := &Document{
		Chat: chat,
		Opts: opts,
		byID: make(map[int64]*telegram.Message, len(chat.Messages)),
	}
	for i := range chat.Messages {
		d.byID[chat.Messages[i].ID] = &chat.Messages[i]
	}

	users := make([]string, 0, len(opts.Users))
	for _, u := range opts.Users {
		if u = strings.TrimSpace(u); u != "" {
			users = append(users, strings.ToLower(u))
		}
	}

	d.Messages = make([]telegram.Message, 0, len(chat.Messages))
	for _, m := range chat.Messages {
		if keep(&m, opts, users) {
			d.Messages = append(d.Messages, m)
		}
	}
	return d
}

func keep(m *telegram.Message, opts Options, users []string) bool {
	if opts.NoService && m.IsService() {
		return false
	}
	if !opts.Since.IsZero() && m.Date.Before(opts.Since) {
		return false
	}
	if !opts.Until.IsZero() && m.Date.After(opts.Until) {
		return false
	}
	if len(users) > 0 &&
		!slices.Contains(users, strings.ToLower(m.Author())) &&
		!slices.Contains(users, strings.ToLower(m.AuthorID())) {
		return false
	}
	if opts.NoMedia && !m.IsService() && m.PlainText() == "" {
		return false
	}
	return true
}

// Lookup returns the original message with the given ID, even if it was
// filtered out, so replies can still quote it.
func (d *Document) Lookup(id int64) (*telegram.Message, bool) {
	m, ok := d.byID[id]
	return m, ok
}

// Format renders a Document to w.
type Format interface {
	// Name is the value accepted by --format.
	Name() string
	// Ext is the output file extension without a dot.
	Ext() string
	Render(w io.Writer, d *Document) error
}

var formats = map[string]Format{}

func register(f Format) { formats[f.Name()] = f }

// LookupFormat returns the format registered under name.
func LookupFormat(name string) (Format, error) {
	f, ok := formats[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("unknown format %q (available: %s)", name, strings.Join(FormatNames(), ", "))
	}
	return f, nil
}

// FormatNames lists registered formats in alphabetical order.
func FormatNames() []string {
	names := make([]string, 0, len(formats))
	for n := range formats {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
