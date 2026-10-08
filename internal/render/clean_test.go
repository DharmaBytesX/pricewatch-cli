package render

import (
	"bytes"
	"testing"
)

func TestClean(t *testing.T) {
	for in, want := range map[string]string{
		"Elden Ring (PS5)":              "Elden Ring (PS5)",
		"Pokémon Écarlate\tFR\n":        "Pokémon Écarlate\tFR\n",
		"Name\x1b]0;owned\x07\x1b[2J":   "Name]0;owned[2J",
		"price\rfake":                   "pricefake",
		"a\u202eevil\u2066b\u0085c\x7f": "aevilbc",
	} {
		if got := Clean(in); got != want {
			t.Errorf("Clean(%q) = %q, want %q", in, got, want)
		}
	}
}

// A character cut between two writes is kept whole; controls are dropped.
func TestForTerminal(t *testing.T) {
	var b bytes.Buffer
	w := ForTerminal(&b)
	name := []byte("Café\x1b[31m rouge\n")
	for _, part := range [][]byte{name[:4], name[4:5], name[5:]} { // "Caf", the first byte of é, the rest
		if _, err := w.Write(part); err != nil {
			t.Fatal(err)
		}
	}
	if got := b.String(); got != "Café[31m rouge\n" {
		t.Fatalf("got %q", got)
	}
}
