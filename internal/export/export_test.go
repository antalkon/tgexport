package export

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/antalkon/tgexport/internal/telegram"
)

var update = flag.Bool("update", false, "rewrite golden files")

func loadChat(t *testing.T, name string) *telegram.Chat {
	t.Helper()
	chats, err := telegram.Load(filepath.Join("../../testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return &chats[0]
}

func render(t *testing.T, format string, d *Document) string {
	t.Helper()
	f, err := LookupFormat(format)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := f.Render(&buf, d); err != nil {
		t.Fatalf("render %s: %v", format, err)
	}
	return buf.String()
}

func TestGolden(t *testing.T) {
	for _, fixture := range []string{"personal", "group"} {
		for _, format := range []string{"txt", "md", "csv", "json"} {
			t.Run(fixture+"."+format, func(t *testing.T) {
				d := NewDocument(loadChat(t, fixture+".json"), Options{})
				got := render(t, format, d)
				golden := filepath.Join("testdata", fixture+".golden."+format)
				if *update {
					if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				want, err := os.ReadFile(golden)
				if err != nil {
					t.Fatalf("%v (run go test ./... -update to create)", err)
				}
				if got != string(want) {
					t.Errorf("output differs from %s:\n--- got ---\n%s\n--- want ---\n%s", golden, got, want)
				}
			})
		}
	}
}

func TestTextLayout(t *testing.T) {
	d := NewDocument(loadChat(t, "personal.json"), Options{
		Layout:     "{date}:{time}:{user} {text}",
		DateFormat: "02.01.2006",
		TimeFormat: "15:04",
	})
	got := strings.SplitN(render(t, "txt", d), "\n", 2)[0]
	if want := "01.10.2026:09:00:Bob Morning!"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMultilineIndent(t *testing.T) {
	d := NewDocument(loadChat(t, "personal.json"), Options{})
	out := render(t, "txt", d)
	if !strings.Contains(out, "Alice: Hi Bob\n    how are you?\n") {
		t.Errorf("continuation line not indented:\n%s", out)
	}
}

func TestFilters(t *testing.T) {
	chat := loadChat(t, "personal.json")
	day := func(s string) time.Time {
		v, _ := time.ParseInLocation("2006-01-02", s, time.Local)
		return v
	}
	tests := []struct {
		name string
		opts Options
		ids  []int64
	}{
		{"all", Options{}, []int64{1, 2, 3, 4, 5, 6}},
		{"by name", Options{Users: []string{"alice"}}, []int64{2, 4}},
		{"by id", Options{Users: []string{"user2002"}}, []int64{1, 3, 5}},
		{"since", Options{Since: day("2026-10-02")}, []int64{4, 5, 6}},
		{"until", Options{Until: day("2026-10-02")}, []int64{1, 2, 3}},
		{"no media", Options{NoMedia: true}, []int64{1, 2, 4, 6}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDocument(chat, tt.opts)
			var ids []int64
			for _, m := range d.Messages {
				ids = append(ids, m.ID)
			}
			if !slices.Equal(ids, tt.ids) {
				t.Errorf("ids = %v, want %v", ids, tt.ids)
			}
		})
	}

	group := loadChat(t, "group.json")
	if d := NewDocument(group, Options{NoService: true}); len(d.Messages) != 4 {
		t.Errorf("no-service kept %d messages, want 4", len(d.Messages))
	}
}

func TestHTML(t *testing.T) {
	d := NewDocument(loadChat(t, "group.json"), Options{Me: "Carol"})
	out := render(t, "html", d)
	for _, want := range []string{
		`<title>Project: team / chat</title>`,
		`<a href="https://example.com/docs" target="_blank" rel="noopener noreferrer">docs</a>`,
		`<strong>run</strong>`,
		`<code>make &lt;test&gt;</code>`,
		`<pre><code data-lang="shell">go test ./...</code></pre>`,
		`Forwarded from News Channel`,
		`class="msg m mine`,
		`[poll Lunch?]`,
		`Dave pinned a message`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("HTML missing %q", want)
		}
	}
	if strings.Contains(out, "javascript:alert") && strings.Contains(out, `href="javascript`) {
		t.Error("unsafe javascript: link rendered as anchor")
	}
	if strings.Contains(out, "http://") || strings.Contains(out, "<link") || strings.Contains(out, "<script src") {
		t.Error("HTML must be self-contained (no external resources)")
	}
}

func TestReplyToMissingMessage(t *testing.T) {
	chat := &telegram.Chat{Name: "x", Messages: []telegram.Message{{ID: 2, Type: "message", From: "A", ReplyTo: 99}}}
	out := render(t, "html", NewDocument(chat, Options{}))
	if !strings.Contains(out, "deleted message") {
		t.Error("reply to unknown message should render as deleted")
	}
}

func TestLookupFormat(t *testing.T) {
	if _, err := LookupFormat("HTML"); err != nil {
		t.Errorf("format names should be case-insensitive: %v", err)
	}
	if _, err := LookupFormat("pdf"); err == nil || !strings.Contains(err.Error(), "available") {
		t.Errorf("unknown format err = %v", err)
	}
	if got := strings.Join(FormatNames(), ","); got != "csv,html,json,md,txt" {
		t.Errorf("FormatNames = %s", got)
	}
}

func TestAnchor(t *testing.T) {
	tests := map[string]string{
		"https://a.b/c":         `<a href="https://a.b/c"`,
		"example.com/x":         `<a href="https://example.com/x"`,
		"tg://resolve?domain=x": `<a href="tg://resolve?domain=x"`,
		"javascript:alert(1)":   "",
		"data:text/html,hi":     "",
	}
	for in, prefix := range tests {
		got := anchor(in, "t")
		if prefix == "" {
			if got != "t" {
				t.Errorf("anchor(%q) = %q, want plain text", in, got)
			}
			continue
		}
		if !strings.HasPrefix(got, prefix) {
			t.Errorf("anchor(%q) = %q, want prefix %q", in, got, prefix)
		}
	}
}
