package export

import (
	"embed"
	"hash/fnv"
	"html"
	"html/template"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/antalkon/tgexport/internal/telegram"
)

//go:embed templates/chat.html.tmpl
var templateFS embed.FS

var chatTemplate = template.Must(template.ParseFS(templateFS, "templates/chat.html.tmpl"))

// groupWindow is how close consecutive messages of one author must be to be
// drawn as a single group (name shown once).
const groupWindow = 5 * time.Minute

// paletteSize must match the number of .c0….cN author colours in the template.
const paletteSize = 8

func init() { register(htmlFormat{}) }

// htmlFormat renders a self-contained HTML page (inline CSS and JS, no
// external requests) that looks like a chat.
type htmlFormat struct{}

func (htmlFormat) Name() string { return "html" }
func (htmlFormat) Ext() string  { return "html" }

type htmlPage struct {
	Title     string
	Type      string
	Count     int
	Period    string
	Generated string
	Days      []htmlDay
}

type htmlDay struct {
	Date     string
	Messages []htmlMessage
}

type htmlMessage struct {
	ID        int64
	Service   bool
	Author    string
	Color     int
	Mine      bool
	ShowName  bool
	Time      string
	FullTime  string
	Edited    string
	Forwarded string
	Reply     *htmlReply
	Media     string
	Body      template.HTML
	Reactions []string
}

type htmlReply struct {
	ID      int64
	Author  string
	Snippet string
}

func (htmlFormat) Render(w io.Writer, d *Document) error {
	page := htmlPage{
		Title:     d.Chat.Title(),
		Type:      strings.ReplaceAll(d.Chat.Type, "_", " "),
		Count:     len(d.Messages),
		Generated: time.Now().Format("2006-01-02 15:04"),
	}
	if n := len(d.Messages); n > 0 {
		page.Period = d.Messages[0].Date.Format("2006-01-02") + " — " + d.Messages[n-1].Date.Format("2006-01-02")
	}

	me := strings.ToLower(strings.TrimSpace(d.Opts.Me))
	var prev *telegram.Message
	for i := range d.Messages {
		m := &d.Messages[i]
		day := m.Date.Format("Monday, 2 January 2006")
		if len(page.Days) == 0 || page.Days[len(page.Days)-1].Date != day {
			page.Days = append(page.Days, htmlDay{Date: day})
			prev = nil
		}
		hm := htmlMessage{
			ID:       m.ID,
			Service:  m.IsService(),
			Author:   m.Author(),
			Color:    colorIndex(m.AuthorID() + m.Author()),
			Time:     m.Date.Format("15:04"),
			FullTime: m.Date.Format("2006-01-02 15:04:05"),
		}
		if me != "" {
			hm.Mine = strings.ToLower(m.Author()) == me || strings.ToLower(m.AuthorID()) == me
		}
		if hm.Service {
			hm.Body = template.HTML(html.EscapeString(m.ServiceText()))
		} else {
			hm.ShowName = prev == nil || prev.IsService() || prev.Author() != m.Author() ||
				m.Date.Sub(prev.Date.Time) > groupWindow || m.ReplyTo != 0
			hm.Body = entitiesHTML(m.Entities())
			hm.Forwarded = m.ForwardedFrom
			if !d.Opts.NoMedia {
				hm.Media = m.MediaLabel()
			}
			if !m.Edited.IsZero() {
				hm.Edited = m.Edited.Format("2006-01-02 15:04:05")
			}
			if m.ReplyTo != 0 {
				hm.Reply = &htmlReply{ID: m.ReplyTo, Author: "deleted message"}
				if r, ok := d.Lookup(m.ReplyTo); ok {
					hm.Reply.Author = r.Author()
					hm.Reply.Snippet = snippet(r.Body(true), 90)
				}
			}
			for _, r := range m.Reactions {
				hm.Reactions = append(hm.Reactions, r.Label()+" "+itoa(int64(r.Count)))
			}
		}
		page.Days[len(page.Days)-1].Messages = append(page.Days[len(page.Days)-1].Messages, hm)
		prev = m
	}
	return chatTemplate.Execute(w, page)
}

func colorIndex(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() % paletteSize)
}

// entitiesHTML converts Telegram text entities to escaped HTML.
func entitiesHTML(text telegram.Text) template.HTML {
	var b strings.Builder
	for _, e := range text {
		t := html.EscapeString(e.Text)
		switch e.Type {
		case "bold":
			b.WriteString("<strong>" + t + "</strong>")
		case "italic":
			b.WriteString("<em>" + t + "</em>")
		case "underline":
			b.WriteString("<u>" + t + "</u>")
		case "strikethrough":
			b.WriteString("<s>" + t + "</s>")
		case "spoiler":
			b.WriteString(`<span class="spoiler" title="spoiler">` + t + "</span>")
		case "code":
			b.WriteString("<code>" + t + "</code>")
		case "pre":
			b.WriteString(`<pre><code`)
			if e.Language != "" {
				b.WriteString(` data-lang="` + html.EscapeString(e.Language) + `"`)
			}
			b.WriteString(">" + t + "</code></pre>")
		case "blockquote", "expandable_blockquote":
			b.WriteString("<blockquote>" + t + "</blockquote>")
		case "link":
			b.WriteString(anchor(e.Text, t))
		case "text_link":
			b.WriteString(anchor(e.Href, t))
		case "email":
			b.WriteString(anchor("mailto:"+e.Text, t))
		case "phone":
			b.WriteString(anchor("tel:"+e.Text, t))
		case "mention":
			b.WriteString(anchor("https://t.me/"+strings.TrimPrefix(e.Text, "@"), t))
		case "hashtag", "cashtag", "bot_command", "mention_name":
			b.WriteString(`<span class="accent">` + t + "</span>")
		default:
			b.WriteString(t)
		}
	}
	return template.HTML(b.String())
}

// anchor returns a link for safe schemes only; anything else (javascript:,
// data:, ...) is rendered as plain text.
func anchor(href, escapedText string) string {
	u, ok := safeURL(href)
	if !ok {
		return escapedText
	}
	return `<a href="` + html.EscapeString(u) + `" target="_blank" rel="noopener noreferrer">` + escapedText + "</a>"
}

// safeURL normalises href and reports whether it uses an allowed scheme.
// Scheme-less links such as "example.com/x" are treated as https.
func safeURL(href string) (string, bool) {
	href = strings.TrimSpace(href)
	u, err := url.Parse(href)
	if err != nil {
		return "", false
	}
	if u.Scheme == "" {
		if u, err = url.Parse("https://" + strings.TrimPrefix(href, "//")); err != nil {
			return "", false
		}
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "tg", "mailto", "tel":
		return u.String(), true
	}
	return "", false
}
