package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/antalkon/tgexport/internal/telegram"
)

// Stats summarises one chat.
type Stats struct {
	Total, Text, Media, Service int
	Authors                     []AuthorCount
	First, Last                 string
	BusiestDay                  string
	BusiestDayCount             int
}

// AuthorCount is the number of non-service messages sent by one author.
type AuthorCount struct {
	Name  string
	Count int
}

// ComputeStats walks the messages of c once and aggregates counters.
func ComputeStats(c *telegram.Chat) Stats {
	var s Stats
	byAuthor := map[string]int{}
	byDay := map[string]int{}
	for i := range c.Messages {
		m := &c.Messages[i]
		s.Total++
		day := m.Date.Format("2006-01-02")
		byDay[day]++
		if s.First == "" || day < s.First {
			s.First = day
		}
		if day > s.Last {
			s.Last = day
		}
		if m.IsService() {
			s.Service++
			continue
		}
		byAuthor[m.Author()]++
		if m.MediaLabel() != "" {
			s.Media++
		}
		if m.PlainText() != "" {
			s.Text++
		}
	}
	for name, n := range byAuthor {
		s.Authors = append(s.Authors, AuthorCount{name, n})
	}
	sort.Slice(s.Authors, func(i, j int) bool {
		if s.Authors[i].Count != s.Authors[j].Count {
			return s.Authors[i].Count > s.Authors[j].Count
		}
		return s.Authors[i].Name < s.Authors[j].Name
	})
	for day, n := range byDay {
		if n > s.BusiestDayCount || (n == s.BusiestDayCount && day < s.BusiestDay) {
			s.BusiestDay, s.BusiestDayCount = day, n
		}
	}
	return s
}

func (a *App) runStats(args []string) error {
	fs := flag.NewFlagSet("stats", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() { fmt.Fprintln(a.Stderr, "Usage: tgexport stats <result.json | export-folder>") }
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return usageError{err.Error()}
	}
	if len(pos) != 1 {
		fs.Usage()
		return usagef("expected exactly one input file")
	}
	chats, err := telegram.Load(pos[0])
	if err != nil {
		return err
	}
	for i := range chats {
		if i > 0 {
			fmt.Fprintln(a.Stdout)
		}
		writeStats(a.Stdout, &chats[i], ComputeStats(&chats[i]))
	}
	return nil
}

func writeStats(w io.Writer, c *telegram.Chat, s Stats) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "Chat:\t%s (%s, id %d)\n", c.Title(), c.Type, c.ID)
	if s.Total > 0 {
		fmt.Fprintf(tw, "Period:\t%s — %s\n", s.First, s.Last)
	}
	fmt.Fprintf(tw, "Messages:\t%d (text %d, media %d, service %d)\n", s.Total, s.Text, s.Media, s.Service)
	if s.BusiestDayCount > 0 {
		fmt.Fprintf(tw, "Busiest day:\t%s (%d messages)\n", s.BusiestDay, s.BusiestDayCount)
	}
	tw.Flush()
	if len(s.Authors) == 0 {
		return
	}
	fmt.Fprintln(w, "Authors:")
	tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', tabwriter.AlignRight)
	for _, a := range s.Authors {
		pct := 100 * float64(a.Count) / float64(s.Total-s.Service)
		fmt.Fprintf(tw, "  %d\t%.1f%%\t  %s\n", a.Count, pct, strings.TrimSpace(a.Name))
	}
	tw.Flush()
}
