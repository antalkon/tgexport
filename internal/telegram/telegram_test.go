package telegram

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

const testdata = "../../testdata"

func load(t *testing.T, name string) []Chat {
	t.Helper()
	chats, err := Load(filepath.Join(testdata, name))
	if err != nil {
		t.Fatalf("Load(%s): %v", name, err)
	}
	return chats
}

func TestLoadPersonal(t *testing.T) {
	chats := load(t, "personal.json")
	if len(chats) != 1 {
		t.Fatalf("got %d chats, want 1", len(chats))
	}
	c := chats[0]
	if c.Name != "Alice" || c.Type != "personal_chat" || len(c.Messages) != 6 {
		t.Fatalf("unexpected chat header: %+v (%d messages)", c.Name, len(c.Messages))
	}
	m := c.Messages[1]
	if got := m.Date.Format(TimeLayout); got != "2026-10-01T09:01:30" {
		t.Errorf("date = %s", got)
	}
	if m.ReplyTo != 1 || m.PlainText() != "Hi Bob\nhow are you?" {
		t.Errorf("message 2 = %+v", m)
	}
	if m.Reactions[0].Label() != "❤" || m.Reactions[0].Count != 1 {
		t.Errorf("reactions = %+v", m.Reactions)
	}
	if got := c.Messages[5].Author(); got != "user3003" {
		t.Errorf("deleted author fallback = %q, want user3003", got)
	}
	if c.Messages[3].Edited.IsZero() {
		t.Error("edited timestamp not parsed")
	}
}

func TestLoadGroupMixedText(t *testing.T) {
	c := load(t, "group.json")[0]
	m := c.Messages[1]
	want := "See docs (https://example.com/docs) and run: make <test> evil (javascript:alert(1))"
	if got := m.PlainText(); got != want {
		t.Errorf("PlainText =\n%q\nwant\n%q", got, want)
	}
	if len(m.Entities()) != 8 {
		t.Errorf("got %d entities, want 8", len(m.Entities()))
	}
}

func TestRichMessageFlatten(t *testing.T) {
	c := load(t, "group.json")[0]
	if got, want := c.Messages[3].PlainText(), "Plan\nShip v1\na | b"; got != want {
		t.Errorf("rich text = %q, want %q", got, want)
	}
}

func TestLoadAccountExport(t *testing.T) {
	chats := load(t, "account.json")
	if len(chats) != 3 {
		t.Fatalf("got %d chats, want 3 (2 chats + 1 left chat)", len(chats))
	}
	if chats[2].Type != "private_supergroup" {
		t.Errorf("left chat type = %s", chats[2].Type)
	}
}

func TestLoadDirectory(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), DefaultFileName) {
		t.Errorf("Load(empty dir) err = %v, want mention of %s", err, DefaultFileName)
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := Parse(strings.NewReader(`{"name":"x"}`)); !errors.Is(err, ErrNoChats) {
		t.Errorf("err = %v, want ErrNoChats", err)
	}
	if _, err := Parse(strings.NewReader(`not json`)); err == nil {
		t.Error("expected decode error")
	}
}

func TestTextUnmarshal(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{`null`, ""},
		{`""`, ""},
		{`"hi"`, "hi"},
		{`["a", {"type":"bold","text":"b"}, "c"]`, "abc"},
	}
	for _, tt := range tests {
		var txt Text
		if err := json.Unmarshal([]byte(tt.in), &txt); err != nil {
			t.Fatalf("%s: %v", tt.in, err)
		}
		m := Message{Text: txt}
		if got := m.PlainText(); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMediaLabel(t *testing.T) {
	tests := []struct {
		name string
		m    Message
		want string
	}{
		{"none", Message{}, ""},
		{"sticker", Message{MediaType: "sticker", StickerEmoji: "👍"}, "[sticker 👍]"},
		{"voice", Message{MediaType: "voice_message", Duration: 75}, "[voice message 1:15]"},
		{"long video", Message{MediaType: "video_file", FileName: "a.mp4", Duration: 3725}, "[video a.mp4 1:02:05]"},
		{"gif", Message{MediaType: "animation"}, "[GIF]"},
		{"photo", Message{Photo: "photos/1.jpg"}, "[photo]"},
		{"file", Message{File: "x", FileName: "doc.pdf"}, "[file doc.pdf]"},
		{"poll", Message{Poll: &Poll{Question: "Lunch?"}}, "[poll Lunch?]"},
		{"contact", Message{Contact: &Contact{FirstName: "Ann", PhoneNumber: "+1"}}, "[contact Ann +1]"},
		{"unknown", Message{MediaType: "new_thing"}, "[new thing]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.MediaLabel(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestServiceText(t *testing.T) {
	c := load(t, "group.json")[0]
	if got := c.Messages[0].ServiceText(); got != "added Dave, Deleted Account" {
		t.Errorf("invite_members = %q", got)
	}
	if got := c.Messages[0].Author(); got != "Carol" {
		t.Errorf("service author = %q, want actor Carol", got)
	}
	if got := (&Message{Action: "some_new_action"}).ServiceText(); got != "some new action" {
		t.Errorf("unknown action = %q", got)
	}
}
