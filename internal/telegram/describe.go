package telegram

import (
	"encoding/json"
	"fmt"
	"strings"
)

// IsService reports whether m is a service message (joins, pins, calls...).
func (m *Message) IsService() bool { return m.Type == TypeService }

// Author returns the display name of whoever sent or triggered the message.
func (m *Message) Author() string {
	name, id := m.From, m.FromID
	if m.IsService() {
		name, id = m.Actor, m.ActorID
	}
	switch {
	case strings.TrimSpace(name) != "":
		return name
	case id != "":
		return id
	default:
		return "Deleted Account"
	}
}

// AuthorID returns the stable sender identifier, e.g. "user123".
func (m *Message) AuthorID() string {
	if m.IsService() {
		return m.ActorID
	}
	return m.FromID
}

// Entities returns the formatted text of the message. Messages sent as
// "rich messages" (articles with headings and tables) have no plain text, so
// their blocks are flattened into plain paragraphs.
func (m *Message) Entities() Text {
	if len(m.Text) > 0 || len(m.RichMessage) == 0 {
		return m.Text
	}
	s := flattenRich(m.RichMessage)
	return plainText(s)
}

// PlainText returns the message text with formatting stripped. Text links
// keep their target in parentheses so no information is lost.
func (m *Message) PlainText() string {
	var b strings.Builder
	for _, e := range m.Entities() {
		b.WriteString(e.Text)
		if e.Type == "text_link" && e.Href != "" && e.Href != e.Text {
			fmt.Fprintf(&b, " (%s)", e.Href)
		}
	}
	return b.String()
}

// MediaLabel describes an attachment in a short bracketed form such as
// "[photo]" or "[voice message 0:02]". It returns "" when there is none.
func (m *Message) MediaLabel() string {
	switch {
	case m.MediaType == "sticker":
		return bracket("sticker", m.StickerEmoji)
	case m.MediaType == "voice_message":
		return bracket("voice message", duration(m.Duration))
	case m.MediaType == "video_message":
		return bracket("video message", duration(m.Duration))
	case m.MediaType == "animation":
		return "[GIF]"
	case m.MediaType == "video_file":
		return bracket("video", m.FileName, duration(m.Duration))
	case m.MediaType == "audio_file":
		return bracket("audio", m.FileName, duration(m.Duration))
	case m.MediaType != "":
		return bracket(strings.ReplaceAll(m.MediaType, "_", " "), m.FileName)
	case m.Photo != "":
		return "[photo]"
	case m.File != "" || m.FileName != "":
		return bracket("file", m.FileName)
	case m.Poll != nil:
		return bracket("poll", m.Poll.Question)
	case m.Location != nil:
		return bracket("location", fmt.Sprintf("%.5f, %.5f", m.Location.Latitude, m.Location.Longitude))
	case m.Contact != nil:
		name := strings.TrimSpace(m.Contact.FirstName + " " + m.Contact.LastName)
		return bracket("contact", name, m.Contact.PhoneNumber)
	}
	return ""
}

// ServiceText describes a service message in English, e.g. "pinned a message".
func (m *Message) ServiceText() string {
	members := m.memberList()
	switch m.Action {
	case "create_group", "create_channel":
		return withDetail("created the chat", quote(m.Title))
	case "invite_members":
		return withDetail("added", members)
	case "remove_members":
		return withDetail("removed", members)
	case "join_group_by_link":
		return "joined the group via invite link"
	case "join_group_by_request":
		return "joined the group by request"
	case "pin_message":
		return "pinned a message"
	case "edit_group_title":
		return withDetail("changed the title to", quote(m.Title))
	case "edit_group_photo":
		return "changed the group photo"
	case "delete_group_photo":
		return "removed the group photo"
	case "migrate_to_supergroup", "migrate_from_group":
		return "converted the group to a supergroup"
	case "phone_call":
		return withDetail("phone call", duration(m.Duration))
	case "group_call":
		return withDetail("started a voice chat", duration(m.Duration))
	case "clear_history":
		return "cleared the history"
	case "":
		return "service message"
	}
	return strings.ReplaceAll(m.Action, "_", " ")
}

func (m *Message) memberList() string {
	names := make([]string, 0, len(m.Members))
	for _, p := range m.Members {
		if p == nil || *p == "" {
			names = append(names, "Deleted Account")
			continue
		}
		names = append(names, *p)
	}
	return strings.Join(names, ", ")
}

// Body returns the full single-string representation used by plain formats:
// media label (if any) followed by the text.
func (m *Message) Body(withMedia bool) string {
	if m.IsService() {
		return m.ServiceText()
	}
	text := m.PlainText()
	if !withMedia {
		return text
	}
	return joinNonEmpty(" ", m.MediaLabel(), text)
}

func bracket(kind string, details ...string) string {
	d := joinNonEmpty(" ", details...)
	if d == "" {
		return "[" + kind + "]"
	}
	return "[" + kind + " " + d + "]"
}

func withDetail(s, detail string) string { return joinNonEmpty(" ", s, detail) }

func quote(s string) string {
	if s == "" {
		return ""
	}
	return "«" + s + "»"
}

func duration(sec int) string {
	if sec <= 0 {
		return ""
	}
	if sec >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", sec/3600, sec%3600/60, sec%60)
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

func joinNonEmpty(sep string, parts ...string) string {
	out := parts[:0:0]
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, sep)
}

// flattenRich extracts readable text from a rich_message tree. The schema is
// open-ended (headings, paragraphs, tables, lists), so instead of modelling
// every block type it walks the tree and joins text leaves: inline nodes are
// concatenated, block nodes are separated by newlines.
func flattenRich(raw json.RawMessage) string {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return ""
	}
	var lines []string
	var walkBlocks func(v any)
	walkBlocks = func(v any) {
		switch n := v.(type) {
		case []any:
			for _, x := range n {
				walkBlocks(x)
			}
		case map[string]any:
			if cells, ok := n["cells"].([]any); ok { // table row → "a | b | c"
				parts := make([]string, 0, len(cells))
				for _, c := range cells {
					parts = append(parts, strings.TrimSpace(inline(c)))
				}
				lines = append(lines, strings.Join(parts, " | "))
				return
			}
			if t, ok := n["text"]; ok {
				if s := strings.TrimSpace(inline(t)); s != "" {
					lines = append(lines, s)
				}
			}
			for _, key := range []string{"blocks", "items", "rows", "cells"} {
				if c, ok := n[key]; ok {
					walkBlocks(c)
				}
			}
		}
	}
	walkBlocks(root)
	return strings.Join(lines, "\n")
}

func inline(v any) string {
	switch n := v.(type) {
	case string:
		return n
	case []any:
		var b strings.Builder
		for _, x := range n {
			b.WriteString(inline(x))
		}
		return b.String()
	case map[string]any:
		if t, ok := n["text"]; ok {
			return inline(t)
		}
	}
	return ""
}
