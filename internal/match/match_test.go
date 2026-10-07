package match

import "testing"

func TestAllWords(t *testing.T) {
	cases := []struct {
		name, query string
		want        bool
	}{
		{"Pokémon Écarlate (Switch)", "pokemon ecarlate", true},
		{"Elden Ring (PS5)", "ps5 elden", true},
		{"Elden Ring (PS5)", "elden ring xbox", false},
		{"iPhone 13 128 Go", "", true},
		{"Œuvre", "OEUVRE", false}, // œ is a letter of its own
	}
	for _, c := range cases {
		if got := AllWords(c.name, c.query); got != c.want {
			t.Errorf("AllWords(%q, %q) = %v", c.name, c.query, got)
		}
	}
}

func TestSame(t *testing.T) {
	if !Same("Elden  Ring (PS5)", "elden ring (ps5)") || Same("Elden Ring", "Elden Ring (PS5)") {
		t.Fatal("Same")
	}
	if Fold("Crème Brûlée") != "creme brulee" {
		t.Fatal(Fold("Crème Brûlée"))
	}
}
