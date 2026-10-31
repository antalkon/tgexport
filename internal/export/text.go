package export

import (
	"bufio"
	"io"
	"strings"

	"github.com/antalkon/tgexport/internal/telegram"
)

// Defaults for the text format.
const (
	DefaultLayout     = "{date} {time} {user}: {text}"
	DefaultDateFormat = "2006-01-02"
	DefaultTimeFormat = "15:04:05"
)

// continuationIndent prefixes the 2nd+ lines of multi-line messages so every
// message still starts at column 0 and stays grep-friendly.
const continuationIndent = "    "

func init() { register(textFormat{}) }

type textFormat struct{}

func (textFormat) Name() string { return "txt" }
func (textFormat) Ext() string  { return "txt" }

func (textFormat) Render(w io.Writer, d *Document) error {
	bw := bufio.NewWriter(w)
	layout := orDefault(d.Opts.Layout, DefaultLayout)
	dateFmt := orDefault(d.Opts.DateFormat, DefaultDateFormat)
	timeFmt := orDefault(d.Opts.TimeFormat, DefaultTimeFormat)

	for i := range d.Messages {
		m := &d.Messages[i]
		r := strings.NewReplacer(
			"{date}", m.Date.Format(dateFmt),
			"{time}", m.Date.Format(timeFmt),
			"{user}", m.Author(),
			"{user_id}", m.AuthorID(),
			"{id}", itoa(m.ID),
			"{text}", indentLines(textBody(m, !d.Opts.NoMedia)),
		)
		if _, err := r.WriteString(bw, layout); err != nil {
			return err
		}
		if err := bw.WriteByte('\n'); err != nil {
			return err
		}
	}
	return bw.Flush()
}

// textBody is the message content for line-oriented formats.
func textBody(m *telegram.Message, withMedia bool) string {
	if m.IsService() {
		return "* " + m.ServiceText()
	}
	body := m.Body(withMedia)
	if m.ForwardedFrom != "" {
		body = "[forwarded from " + m.ForwardedFrom + "] " + body
	}
	return body
}

func indentLines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\n", "\n"+continuationIndent)
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
