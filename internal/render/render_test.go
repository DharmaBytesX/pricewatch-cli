package render

import (
	"bytes"
	"testing"
	"time"
)

func TestEuros(t *testing.T) {
	for cents, want := range map[int64]string{
		0: "0,00 €", 5: "0,05 €", 4499: "44,99 €", 129999: "1 299,99 €", 123456789: "1 234 567,89 €", -250: "-2,50 €",
	} {
		if got := Euros(cents); got != want {
			t.Errorf("Euros(%d) = %q, want %q", cents, got, want)
		}
	}
	if OptionalEuros(nil) != "—" {
		t.Error("no price")
	}
}

func TestAgo(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	at := func(d time.Duration) *time.Time { t := now.Add(-d); return &t }
	cases := []struct {
		t    *time.Time
		want string
	}{
		{nil, "never"},
		{at(12 * time.Second), "12s ago"},
		{at(-time.Second), "0s ago"}, // a server clock a little ahead
		{at(5 * time.Minute), "5m ago"},
		{at(3 * time.Hour), "3h ago"},
		{at(50 * time.Hour), "2d ago"},
	}
	for _, c := range cases {
		if got := Ago(c.t, now); got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
}

func TestCondition(t *testing.T) {
	for in, want := range map[string]string{"new": "New", "used": "Used", "any": "Used & new", "other": "other"} {
		if got := Condition(in); got != want {
			t.Errorf("Condition(%q) = %q", in, got)
		}
	}
}

func TestTable(t *testing.T) {
	var b bytes.Buffer
	tb := NewTable(&b, "STORE", "PRICE")
	tb.Row("Amazon FR", "1 299,99 €")
	tb.Row("Fnac", "9,99 €")
	if err := tb.Flush(); err != nil {
		t.Fatal(err)
	}
	want := "STORE      PRICE\nAmazon FR  1 299,99 €\nFnac       9,99 €\n"
	if b.String() != want {
		t.Fatalf("got\n%q\nwant\n%q", b.String(), want)
	}
}
