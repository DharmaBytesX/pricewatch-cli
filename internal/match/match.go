// Package match compares product names the way the Pricewatch website
// does: case and accents ignored, every word of the query present.
package match

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Fold lowercases s and removes accents: "Pokémon" becomes "pokemon".
func Fold(s string) string {
	t := transform.Chain(norm.NFKD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, s)
	if err != nil {
		out = s
	}
	return strings.ToLower(out)
}

// Words returns the folded words of s.
func Words(s string) []string { return strings.Fields(Fold(s)) }

// AllWords reports whether name contains every word of query, in any order.
// An empty query matches every name.
func AllWords(name, query string) bool {
	n := Fold(name)
	for _, w := range Words(query) {
		if !strings.Contains(n, w) {
			return false
		}
	}
	return true
}

// Same reports whether two names are equal, ignoring case, accents and
// spaces between words.
func Same(a, b string) bool {
	return strings.Join(Words(a), " ") == strings.Join(Words(b), " ")
}
