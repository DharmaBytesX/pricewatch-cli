// Package render formats prices, times and tables for the terminal, the
// way the Pricewatch website writes them.
package render

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
	"unicode/utf8"
)

// Euros writes a price in cents the French way: "1 299,99 €".
func Euros(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	whole := strconv.FormatInt(cents/100, 10)
	// thousands separated by spaces
	var b strings.Builder
	for i, r := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return fmt.Sprintf("%s%s,%02d €", sign, b.String(), cents%100)
}

// OptionalEuros writes a price, or "—" when there is none.
func OptionalEuros(cents *int64) string {
	if cents == nil {
		return "—"
	}
	return Euros(*cents)
}

// Ago writes how long ago t was: "12s ago", "5m ago", "3h ago", "2d ago",
// or "never" for nil.
func Ago(t *time.Time, now time.Time) string {
	if t == nil {
		return "never"
	}
	d := now.Sub(*t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

// Date writes a day: "7 Oct 2026".
func Date(t time.Time) string { return t.Local().Format("2 Jan 2006") }

// Condition writes a product condition as the website does.
func Condition(c string) string {
	switch c {
	case "new":
		return "New"
	case "used":
		return "Used"
	case "any":
		return "Used & new"
	}
	return c
}

// Table writes aligned columns. Write rows with Row, then call Flush.
type Table struct{ tw *tabwriter.Writer }

// NewTable starts a table with a header row.
func NewTable(w io.Writer, header ...string) *Table {
	t := &Table{tw: tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)}
	t.Row(header...)
	return t
}

// Row adds a row.
func (t *Table) Row(cells ...string) {
	_, _ = fmt.Fprintln(t.tw, strings.Join(cells, "\t"))
}

// Flush writes the table.
func (t *Table) Flush() error { return t.tw.Flush() }

// Clean drops the characters of s that would let text from the server
// control a terminal: the C0 controls but tab and newline (ESC starts the
// escape sequences that move the cursor, set the window title or clear the
// screen; a carriage return lets a line overwrite another), DEL, the C1
// controls, and the bidirectional overrides that reorder what is shown.
func Clean(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t' || r == '\n':
			return r
		case r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f):
			return -1
		case (r >= 0x202a && r <= 0x202e) || (r >= 0x2066 && r <= 0x2069):
			return -1
		}
		return r
	}, s)
}

// ForTerminal returns a writer to w that cleans what it writes (see Clean).
// The CLI writes through it when its output is a terminal, so a product
// name or a server message cannot drive the terminal; piped output is left
// as it is.
func ForTerminal(w io.Writer) io.Writer { return &termWriter{w: w} }

type termWriter struct {
	w    io.Writer
	rest []byte // the start of a UTF-8 character cut by the last write
}

func (t *termWriter) Write(p []byte) (int, error) {
	b := make([]byte, 0, len(t.rest)+len(p))
	b = append(append(b, t.rest...), p...)
	cut := len(b)
	for i := len(b) - 1; i >= 0 && i >= len(b)-utf8.UTFMax; i-- {
		if utf8.RuneStart(b[i]) {
			if !utf8.FullRune(b[i:]) {
				cut = i
			}
			break
		}
	}
	t.rest = append([]byte(nil), b[cut:]...)
	if _, err := io.WriteString(t.w, Clean(string(b[:cut]))); err != nil {
		return 0, err
	}
	return len(p), nil
}
