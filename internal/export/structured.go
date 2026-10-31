package export

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/antalkon/tgexport/internal/telegram"
)

func init() {
	register(markdownFormat{})
	register(csvFormat{})
	register(jsonFormat{})
}

// markdownFormat renders the chat as a readable Markdown document with one
// section per day.
type markdownFormat struct{}

func (markdownFormat) Name() string { return "md" }
func (markdownFormat) Ext() string  { return "md" }

func (markdownFormat) Render(w io.Writer, d *Document) error {
	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, "# %s\n\n", mdEscape(d.Chat.Title()))
	fmt.Fprintf(bw, "_%s · %d messages_\n", d.Chat.Type, len(d.Messages))

	var day string
	for i := range d.Messages {
		m := &d.Messages[i]
		if dd := m.Date.Format("2006-01-02"); dd != day {
			day = dd
			fmt.Fprintf(bw, "\n## %s\n", day)
		}
		if m.IsService() {
			fmt.Fprintf(bw, "\n_%s · %s %s_\n", m.Date.Format("15:04"), mdEscape(m.Author()), m.ServiceText())
			continue
		}
		fmt.Fprintf(bw, "\n**%s** · %s", mdEscape(m.Author()), m.Date.Format("15:04"))
		if m.ForwardedFrom != "" {
			fmt.Fprintf(bw, " · forwarded from %s", mdEscape(m.ForwardedFrom))
		}
		bw.WriteString("  \n") // hard break: keep the header on its own line
		if m.ReplyTo != 0 {
			if r, ok := d.Lookup(m.ReplyTo); ok {
				fmt.Fprintf(bw, "> ↩ %s: %s\n", mdEscape(r.Author()), mdEscape(snippet(r.Body(true), 80)))
			}
		}
		if label := m.MediaLabel(); label != "" && !d.Opts.NoMedia {
			fmt.Fprintf(bw, "%s  \n", mdEscape(label))
		}
		if text := mdEntities(m.Entities()); text != "" {
			fmt.Fprintf(bw, "%s\n", text)
		}
		if len(m.Reactions) > 0 {
			fmt.Fprintf(bw, "%s\n", reactionsLine(m.Reactions))
		}
	}
	return bw.Flush()
}

// csvFormat renders one row per message, for spreadsheets and scripts.
type csvFormat struct{}

func (csvFormat) Name() string { return "csv" }
func (csvFormat) Ext() string  { return "csv" }

func (csvFormat) Render(w io.Writer, d *Document) error {
	cw := csv.NewWriter(w)
	header := []string{"id", "date", "time", "from", "from_id", "type", "reply_to", "forwarded_from", "media", "text", "reactions"}
	if err := cw.Write(header); err != nil {
		return err
	}
	for i := range d.Messages {
		m := &d.Messages[i]
		text := m.PlainText()
		if m.IsService() {
			text = m.ServiceText()
		}
		media := m.MediaLabel()
		if d.Opts.NoMedia {
			media = ""
		}
		reply := ""
		if m.ReplyTo != 0 {
			reply = itoa(m.ReplyTo)
		}
		row := []string{
			itoa(m.ID), m.Date.Format("2006-01-02"), m.Date.Format("15:04:05"),
			m.Author(), m.AuthorID(), m.Type, reply, m.ForwardedFrom, media, text,
			reactionsLine(m.Reactions),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// jsonFormat writes a simplified, stable JSON structure that is much easier
// to consume than Telegram's raw export.
type jsonFormat struct{}

func (jsonFormat) Name() string { return "json" }
func (jsonFormat) Ext() string  { return "json" }

type jsonMessage struct {
	ID            int64          `json:"id"`
	Date          telegram.Time  `json:"date"`
	Edited        *telegram.Time `json:"edited,omitempty"`
	From          string         `json:"from"`
	FromID        string         `json:"from_id,omitempty"`
	Service       bool           `json:"service,omitempty"`
	ReplyTo       int64          `json:"reply_to,omitempty"`
	ForwardedFrom string         `json:"forwarded_from,omitempty"`
	Media         string         `json:"media,omitempty"`
	Text          string         `json:"text"`
	Reactions     map[string]int `json:"reactions,omitempty"`
}

type jsonChat struct {
	ID       int64         `json:"id"`
	Name     string        `json:"name"`
	Type     string        `json:"type"`
	Count    int           `json:"count"`
	Messages []jsonMessage `json:"messages"`
}

func (jsonFormat) Render(w io.Writer, d *Document) error {
	out := jsonChat{
		ID: d.Chat.ID, Name: d.Chat.Title(), Type: d.Chat.Type,
		Count: len(d.Messages), Messages: make([]jsonMessage, 0, len(d.Messages)),
	}
	for i := range d.Messages {
		m := &d.Messages[i]
		jm := jsonMessage{
			ID: m.ID, Date: m.Date, From: m.Author(), FromID: m.AuthorID(),
			Service: m.IsService(), ReplyTo: m.ReplyTo, ForwardedFrom: m.ForwardedFrom,
			Text: m.PlainText(),
		}
		if !m.Edited.IsZero() {
			e := m.Edited
			jm.Edited = &e
		}
		if m.IsService() {
			jm.Text = m.ServiceText()
		}
		if !d.Opts.NoMedia {
			jm.Media = m.MediaLabel()
		}
		if len(m.Reactions) > 0 {
			jm.Reactions = make(map[string]int, len(m.Reactions))
			for _, r := range m.Reactions {
				jm.Reactions[r.Label()] += r.Count
			}
		}
		out.Messages = append(out.Messages, jm)
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func reactionsLine(rs []telegram.Reaction) string {
	parts := make([]string, 0, len(rs))
	for _, r := range rs {
		parts = append(parts, fmt.Sprintf("%s %d", r.Label(), r.Count))
	}
	return strings.Join(parts, " · ")
}

func snippet(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// mdReplacer escapes the characters that change meaning inline. Square
// brackets are left alone: without a following "(" they are not links.
var mdReplacer = strings.NewReplacer(
	`\`, `\\`, "*", `\*`, "_", `\_`, "`", "\\`", "~", `\~`, "<", "&lt;", ">", "&gt;",
)

func mdEscape(s string) string { return mdReplacer.Replace(s) }

// mdHardBreaks escapes s and keeps its line breaks visible in Markdown.
func mdHardBreaks(s string) string {
	return strings.ReplaceAll(mdEscape(s), "\n", "  \n")
}

// mdEntities converts Telegram text entities to Markdown.
func mdEntities(text telegram.Text) string {
	var b strings.Builder
	for _, e := range text {
		if strings.TrimSpace(e.Text) == "" {
			b.WriteString(mdHardBreaks(e.Text))
			continue
		}
		switch e.Type {
		case "bold":
			b.WriteString("**" + mdHardBreaks(e.Text) + "**")
		case "italic":
			b.WriteString("_" + mdHardBreaks(e.Text) + "_")
		case "strikethrough":
			b.WriteString("~~" + mdHardBreaks(e.Text) + "~~")
		case "code":
			b.WriteString("`` " + e.Text + " ``")
		case "pre":
			b.WriteString("\n```" + e.Language + "\n" + strings.TrimRight(e.Text, "\n") + "\n```\n")
		case "link", "email":
			b.WriteString("<" + e.Text + ">")
		case "text_link":
			if u, ok := safeURL(e.Href); ok {
				b.WriteString("[" + mdHardBreaks(e.Text) + "](" + strings.ReplaceAll(u, ")", "%29") + ")")
			} else {
				b.WriteString(mdHardBreaks(e.Text))
			}
		default:
			b.WriteString(mdHardBreaks(e.Text))
		}
	}
	return b.String()
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
