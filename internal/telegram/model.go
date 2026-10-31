// Package telegram models the JSON produced by Telegram Desktop's
// "Export chat history" feature (machine-readable JSON format).
package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// TimeLayout is the timestamp layout used by Telegram Desktop exports.
// Timestamps are written in the exporting machine's local time without a zone.
const TimeLayout = "2006-01-02T15:04:05"

// Message types.
const (
	TypeMessage = "message"
	TypeService = "service"
)

// Chat is a single exported dialog: a personal chat, a group or a channel.
type Chat struct {
	ID       int64     `json:"id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Messages []Message `json:"messages"`
}

// Title returns the chat name, falling back to a synthetic one for chats
// without a name (e.g. "Saved Messages" or deleted accounts).
func (c *Chat) Title() string {
	if strings.TrimSpace(c.Name) != "" {
		return c.Name
	}
	return fmt.Sprintf("chat_%d", c.ID)
}

// Message is one entry of Chat.Messages. Only the fields tgexport renders are
// decoded; everything else in the export is ignored.
type Message struct {
	ID      int64     `json:"id"`
	Type    string    `json:"type"`
	Date    Time      `json:"date"`
	Edited  Time      `json:"edited"`
	From    string    `json:"from"`
	FromID  string    `json:"from_id"`
	Actor   string    `json:"actor"`
	ActorID string    `json:"actor_id"`
	Action  string    `json:"action"`
	Title   string    `json:"title"`
	Members []*string `json:"members"`
	ReplyTo int64     `json:"reply_to_message_id"`

	ForwardedFrom string `json:"forwarded_from"`
	ViaBot        string `json:"via_bot"`

	Photo        string `json:"photo"`
	File         string `json:"file"`
	FileName     string `json:"file_name"`
	MediaType    string `json:"media_type"`
	MimeType     string `json:"mime_type"`
	StickerEmoji string `json:"sticker_emoji"`
	Duration     int    `json:"duration_seconds"`

	Poll     *Poll     `json:"poll"`
	Location *Location `json:"location_information"`
	Contact  *Contact  `json:"contact_information"`

	Text        Text            `json:"text"`
	RichMessage json.RawMessage `json:"rich_message"`
	Reactions   []Reaction      `json:"reactions"`
}

// Poll is an attached poll.
type Poll struct {
	Question string `json:"question"`
	Closed   bool   `json:"closed"`
	Answers  []struct {
		Text   string `json:"text"`
		Voters int    `json:"voters"`
	} `json:"answers"`
}

// Location is a shared geo point.
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Contact is a shared contact card.
type Contact struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	PhoneNumber string `json:"phone_number"`
}

// Reaction is an aggregated reaction under a message.
type Reaction struct {
	Type     string `json:"type"`
	Count    int    `json:"count"`
	Emoji    string `json:"emoji"`
	Document string `json:"document_id"`
}

// Label returns a printable form of the reaction (emoji or a placeholder for
// custom/paid reactions).
func (r Reaction) Label() string {
	switch {
	case r.Emoji != "":
		return r.Emoji
	case r.Type == "paid":
		return "⭐"
	default:
		return "★"
	}
}

// Entity is a formatted fragment of message text.
type Entity struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Href     string `json:"href,omitempty"`
	Language string `json:"language,omitempty"`
	UserID   int64  `json:"user_id,omitempty"`
}

// Text is message text. Telegram writes it either as a plain string or as an
// array mixing plain strings and entity objects; both decode to entities.
type Text []Entity

// UnmarshalJSON implements json.Unmarshaler.
func (t *Text) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	switch {
	case len(data) == 0 || bytes.Equal(data, []byte("null")):
		*t = nil
		return nil
	case data[0] == '"':
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*t = plainText(s)
		return nil
	}

	var parts []json.RawMessage
	if err := json.Unmarshal(data, &parts); err != nil {
		return fmt.Errorf("text: %w", err)
	}
	out := make(Text, 0, len(parts))
	for _, p := range parts {
		p = bytes.TrimSpace(p)
		if len(p) > 0 && p[0] == '"' {
			var s string
			if err := json.Unmarshal(p, &s); err != nil {
				return err
			}
			out = append(out, plainText(s)...)
			continue
		}
		var e Entity
		if err := json.Unmarshal(p, &e); err != nil {
			return fmt.Errorf("text entity: %w", err)
		}
		out = append(out, e)
	}
	*t = out
	return nil
}

func plainText(s string) Text {
	if s == "" {
		return nil
	}
	return Text{{Type: "plain", Text: s}}
}

// Time is a Telegram export timestamp.
type Time struct{ time.Time }

// UnmarshalJSON implements json.Unmarshaler.
func (t *Time) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil || s == "" {
		t.Time = time.Time{}
		return nil
	}
	v, err := time.ParseInLocation(TimeLayout, s, time.Local)
	if err != nil {
		return fmt.Errorf("parse time %q: %w", s, err)
	}
	t.Time = v
	return nil
}

// MarshalJSON implements json.Marshaler.
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Format(TimeLayout))
}
