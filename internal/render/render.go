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
