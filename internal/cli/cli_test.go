package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/DharmaBytesX/pricewatch-cli/internal/api"
	"github.com/DharmaBytesX/pricewatch-cli/internal/config"
	"github.com/DharmaBytesX/pricewatch-cli/internal/fakeapi"
)

// now is the tests' clock.
var now = time.Date(2026, 10, 7, 15, 0, 0, 0, time.UTC)

// harness runs commands against a fake Pricewatch server.
type harness struct {
	t        *testing.T
	srv      *fakeapi.Server
	env      map[string]string
	stdin    string
	terminal bool
	secret   string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	srv := fakeapi.New()
	t.Cleanup(srv.Close)
	return &harness{t: t, srv: srv, env: map[string]string{
		config.EnvConfig: filepath.Join(t.TempDir(), "pricewatch", "config.json"),
		config.EnvURL:    srv.URL,
	}}
}

// signedIn makes the commands use the token, as after "auth login".
func (h *harness) signedIn(token string) *harness {
	h.env[config.EnvToken] = token
	return h
}

func (h *harness) run(args ...string) (stdout, stderr string, code int) {
	h.t.Helper()
	var out, errOut bytes.Buffer
	app := &App{
		In:              strings.NewReader(h.stdin),
		Out:             &out,
		Err:             &errOut,
		Getenv:          func(k string) string { return h.env[k] },
		Now:             func() time.Time { return now },
		StdinIsTerminal: func() bool { return h.terminal },
		ReadSecret:      func() (string, error) { return h.secret, nil },
		PollInterval:    time.Millisecond,
	}
	code = Execute(app, args)
	return out.String(), errOut.String(), code
}

func expect(t *testing.T, got, want int, stdout, stderr string) {
	t.Helper()
	if got != want {
		t.Fatalf("exit code %d, want %d\nstdout: %s\nstderr: %s", got, want, stdout, stderr)
	}
}

func mustContain(t *testing.T, s string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if !strings.Contains(s, p) {
			t.Fatalf("missing %q in:\n%s", p, s)
		}
	}
}

// ---------------------------------------------------------------- auth

func TestLoginStatusLogout(t *testing.T) {
	h := newHarness(t)
	h.stdin = fakeapi.WriteToken + "\n"
	out, errOut, code := h.run("auth", "login", "--with-token")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "Signed in to "+h.srv.URL+" as lex@example.com (Premium plan)", "Token “laptop”: reads and adds products, expires on 6 Jan 2027")

	path := h.env[config.EnvConfig]
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("config file mode %v: the token must be readable by its owner only", info.Mode().Perm())
	}
	f, _ := config.Load(path)
	if f.Token != fakeapi.WriteToken || f.URL != "" {
		t.Fatalf("saved %+v", f)
	}

	out, errOut, code = h.run("auth", "status")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, h.srv.URL+" (environment)", "lex@example.com (Premium plan: any number of products, checked every minute)",
		"“laptop” (config file): reads and adds products", "6 Jan 2027 (in 91 days)")

	out, errOut, code = h.run("auth", "logout")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "Removed the token from "+path)
	_, errOut, code = h.run("auth", "status")
	expect(t, code, ExitDenied, "", errOut)
	mustContain(t, errOut, "not signed in")
}

func TestLoginWithPromptAndURLFlag(t *testing.T) {
	h := newHarness(t)
	delete(h.env, config.EnvURL)
	h.terminal, h.secret = true, "  "+fakeapi.ReadToken+"  "
	out, errOut, code := h.run("auth", "login", "--url", h.srv.URL+"/")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, errOut, "Paste an API token")
	mustContain(t, out, "Token “dashboard”: reads products and prices")
	if f, _ := config.Load(h.env[config.EnvConfig]); f.URL != h.srv.URL || f.Token != fakeapi.ReadToken {
		t.Fatalf("saved %+v: the address given with --url is kept, without its trailing slash", f)
	}
	// the saved address is used afterwards
	out, errOut, code = h.run("auth", "status")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, h.srv.URL+" (config file)")
	mustContain(t, errOut, "expires soon") // 10 Oct 2026
}

func TestLoginRefusals(t *testing.T) {
	h := newHarness(t)
	h.stdin = "not-a-token\n"
	_, errOut, code := h.run("auth", "login", "--with-token")
	expect(t, code, ExitUsage, "", errOut)
	mustContain(t, errOut, "start with pwt_")

	h.stdin = "pwt_" + strings.Repeat("0", 48)
	_, errOut, code = h.run("auth", "login", "--with-token")
	expect(t, code, ExitDenied, "", errOut)
	mustContain(t, errOut, "unknown, expired or revoked")

	_, errOut, code = h.run("auth", "login") // no terminal
	expect(t, code, ExitUsage, "", errOut)
	mustContain(t, errOut, "--with-token")
}

func TestNotSignedIn(t *testing.T) {
	_, errOut, code := newHarness(t).run("prices")
	expect(t, code, ExitDenied, "", errOut)
	mustContain(t, errOut, `run "pricewatch auth login", or set PRICEWATCH_TOKEN`)
}

func TestRevokedTokenExplainsWhatToDo(t *testing.T) {
	h := newHarness(t).signedIn("pwt_" + strings.Repeat("9", 48))
	_, errOut, code := h.run("prices")
	expect(t, code, ExitDenied, "", errOut)
	mustContain(t, errOut, "invalid, expired or revoked API token", "Settings page")
}

// ---------------------------------------------------------------- prices

func cents(v int64) *int64 { return &v }

func seedProducts(srv *fakeapi.Server) {
	checked := now.Add(-90 * time.Second)
	srv.Products = []api.Product{
		{ID: "11111111-aaaa", Name: "iPhone 13 128 Go", Condition: "any", TargetPriceCents: cents(45050), Watches: []api.Watch{
			{Marketplace: "amazon", URL: "https://amazon.example/i13", Active: true, LastCheckedAt: &checked,
				Latest: &api.Snapshot{PriceCents: cents(42623), InStock: true}},
			{Marketplace: "cdiscount", URL: "https://cdiscount.example/i13", Active: true, LastCheckedAt: &checked,
				Latest: &api.Snapshot{PriceCents: cents(23999), InStock: true}},
			{Marketplace: "micromania", URL: "https://micromania.example/i13", Active: true, LastCheckedAt: &checked,
				Latest: &api.Snapshot{PriceCents: cents(19999), InStock: false}},
			{Marketplace: "fnac", URL: "https://fnac.example/i13", Active: true},
		}},
		{ID: "22222222-bbbb", Name: "Pokémon Écarlate (Switch)", Condition: "new", Watches: []api.Watch{
			{Marketplace: "amazon", URL: "https://amazon.example/pk", Active: true, LastCheckedAt: &checked,
				Latest: &api.Snapshot{PriceCents: cents(129900), InStock: false}},
		}},
	}
}

func TestPricesSummary(t *testing.T) {
	h := newHarness(t).signedIn(fakeapi.ReadToken)
	seedProducts(h.srv)
	out, errOut, code := h.run("prices")
	expect(t, code, ExitOK, out, errOut)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("summary:\n%s", out)
	}
	mustContain(t, lines[0], "PRODUCT", "LOWEST", "STORE", "IN STOCK", "CHECKED")
	mustContain(t, lines[1], "iPhone 13 128 Go", "239,99 €", "Cdiscount", "2 of 4 stores", "1m ago")
	mustContain(t, lines[2], "Pokémon Écarlate (Switch)", "1 299,00 € (out of stock)", "Amazon FR", "0 of 1 stores")
}

func TestPricesDetail(t *testing.T) {
	h := newHarness(t).signedIn(fakeapi.ReadToken)
	seedProducts(h.srv)
	out, errOut, code := h.run("prices", "IPHONE", "128")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "iPhone 13 128 Go\nUsed & new · alerts at or below 450,50 € · lowest in stock 239,99 € at Cdiscount")
	// in stock by price, then out of stock, then not checked yet
	order := []string{"Cdiscount ", "Amazon FR ", "Micromania ", "fnac "}
	last := -1
	for _, store := range order {
		i := strings.Index(out, store)
		if i < last {
			t.Fatalf("%q is out of order:\n%s", store, out)
		}
		last = i
	}
	mustContain(t, out, "199,99 €  out of stock", "checking…", "https://cdiscount.example/i13")

	out, _, code = h.run("prices", "iphone", "--in-stock")
	expect(t, code, ExitOK, out, "")
	if strings.Contains(out, "Micromania") || strings.Contains(out, "fnac") {
		t.Fatalf("--in-stock shows other stores:\n%s", out)
	}

	out, _, code = h.run("prices", "pokemon ecarlate") // accents ignored
	expect(t, code, ExitOK, out, "")
	mustContain(t, out, "no store has it in stock")

	out, _, code = h.run("prices", "pokemon", "--in-stock")
	expect(t, code, ExitOK, out, "")
	mustContain(t, out, "No store to show.")

	out, _, code = h.run("prices", "22222222") // ID prefix
	expect(t, code, ExitOK, out, "")
	mustContain(t, out, "Pokémon Écarlate (Switch)")
}

func TestPricesJSONAndNotFound(t *testing.T) {
	h := newHarness(t).signedIn(fakeapi.ReadToken)
	seedProducts(h.srv)
	out, errOut, code := h.run("prices", "iphone", "--json")
	expect(t, code, ExitOK, out, errOut)
	var got []api.Product
	if err := json.Unmarshal([]byte(out), &got); err != nil || len(got) != 1 || len(got[0].Watches) != 4 {
		t.Fatalf("JSON %v:\n%s", err, out)
	}

	_, errOut, code = h.run("prices", "zelda")
	expect(t, code, ExitNotFound, "", errOut)
	mustContain(t, errOut, `you track no product matching “zelda”; to track it: pricewatch add "zelda"`)
}

func TestPricesSummaryOfANewProduct(t *testing.T) {
	h := newHarness(t).signedIn(fakeapi.ReadToken)
	h.srv.Products = []api.Product{{ID: "p1", Name: "Astro Bot PS5", Watches: []api.Watch{{Marketplace: "amazon", Active: true}}}}
	out, errOut, code := h.run("prices")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "Astro Bot PS5", "0 of 1 stores", "checking…")
}

func TestPricesWithNothingTracked(t *testing.T) {
	out, errOut, code := newHarness(t).signedIn(fakeapi.ReadToken).run("prices")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "You track no product yet")
}

// ---------------------------------------------------------------- add

func seedCatalog(srv *fakeapi.Server) {
	offers := func(id string) []api.CatalogOffer {
		return []api.CatalogOffer{
			{ID: id + "-1", Marketplace: "amazon", URL: "https://amazon.example/" + id},
			{ID: id + "-2", Marketplace: "cdiscount", URL: "https://cdiscount.example/" + id},
		}
	}
	srv.Catalog = []api.CatalogProduct{
		{ID: "cat-elden-ps5", Name: "Elden Ring (PS5)", Type: "video_game", Offers: offers("er5")},
		{ID: "cat-elden-xsx", Name: "Elden Ring (Xbox Series X)", Type: "video_game", Offers: offers("erx")},
		{ID: "cat-iphone", Name: "iPhone 13 128 Go", Type: "smartphone", Offers: offers("i13")},
	}
}

// lastPost returns the last POST request the server received.
func lastPost(srv *fakeapi.Server) string {
	log := srv.Log()
	for i := len(log) - 1; i >= 0; i-- {
		if strings.HasPrefix(log[i], "POST ") {
			return log[i]
		}
	}
	return ""
}

func TestAddExactCatalogMatch(t *testing.T) {
	h := newHarness(t).signedIn(fakeapi.WriteToken)
	seedCatalog(h.srv)
	out, errOut, code := h.run("add", "iphone 13", "128 GO", "--condition", "any", "--max-price", "449,90")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "Tracking iPhone 13 128 Go in 2 stores (Used & new, alerts at or below 449,90 €).",
		`pricewatch prices "iPhone 13 128 Go"`)
	mustContain(t, lastPost(h.srv), "POST /api/v1/catalog/cat-iphone/track", `"condition":"any"`, `"targetPriceCents":44990`)
}

func TestAddAmbiguousName(t *testing.T) {
	h := newHarness(t).signedIn(fakeapi.WriteToken)
	seedCatalog(h.srv)

	// Without a terminal: list the products and stop.
	_, errOut, code := h.run("add", "elden ring")
	expect(t, code, ExitUsage, "", errOut)
	mustContain(t, errOut, "“elden ring” matches 2 catalog products:", "Elden Ring (PS5)  (--id cat-elden-ps5)", "--new")
	if r := lastPost(h.srv); r != "" {
		t.Fatalf("tracked without a choice: %s", r)
	}

	// --yes takes the best match.
	out, errOut, code := h.run("add", "elden ring", "--yes")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, lastPost(h.srv), "/catalog/cat-elden-ps5/track")

	// In a terminal: ask.
	h.terminal, h.stdin = true, "7\n2\n"
	out, errOut, code = h.run("add", "elden ring")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, errOut, "1) Elden Ring (PS5) (2 stores)", "0) none of these", "Type a number from 0 to 2.")
	mustContain(t, lastPost(h.srv), "/catalog/cat-elden-xsx/track")
}

func TestAddFindsAProductInTheStores(t *testing.T) {
	h := newHarness(t).signedIn(fakeapi.WriteToken)
	seedCatalog(h.srv)
	h.srv.DiscoveryPolls = 2
	h.srv.Discover = func(string) *api.CatalogProduct {
		return &api.CatalogProduct{ID: "cat-mkw", Name: "Mario Kart World (Switch 2)", Offers: []api.CatalogOffer{
			{Marketplace: "amazon", URL: "https://amazon.example/mkw"},
			{Marketplace: "micromania", URL: "https://micromania.example/mkw"},
			{Marketplace: "cdiscount", URL: "https://cdiscount.example/mkw"},
		}}
	}
	out, errOut, code := h.run("add", "Mario Kart World Switch 2", "--type", "video_game")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, errOut, "Searching the stores for “Mario Kart World Switch 2”…", "Found in 3 stores.")
	mustContain(t, out, "Tracking Mario Kart World (Switch 2) in 3 stores (New).")
	if got := h.srv.DiscoveryType("Mario Kart World Switch 2"); got != "video_game" {
		t.Fatalf("type sent: %q", got)
	}

	// --new searches the stores even when the catalog has matches.
	h.srv.Discover = func(string) *api.CatalogProduct { return nil }
	_, errOut, code = h.run("add", "elden ring", "--new")
	expect(t, code, ExitNotFound, "", errOut)
	mustContain(t, errOut, "no store sells “elden ring” right now")
}

func TestAddWaitsForTheFirstPrices(t *testing.T) {
	h := newHarness(t).signedIn(fakeapi.WriteToken)
	seedCatalog(h.srv)
	h.srv.CheckAfterPolls = 3
	out, errOut, code := h.run("add", "iPhone 13 128 Go", "--wait")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "Tracking iPhone 13 128 Go in 2 stores (New).", "lowest in stock 100,00 € at Amazon FR",
		"Amazon FR  100,00 €  in stock", "Cdiscount  110,00 €  out of stock")

	h.srv.Products = nil
	out, errOut, code = h.run("add", "iPhone 13 128 Go", "--json")
	expect(t, code, ExitOK, out, errOut)
	var p api.Product
	if err := json.Unmarshal([]byte(out), &p); err != nil || p.Name != "iPhone 13 128 Go" {
		t.Fatalf("JSON %v:\n%s", err, out)
	}
}

func TestAddRefusals(t *testing.T) {
	h := newHarness(t).signedIn(fakeapi.ReadToken)
	seedCatalog(h.srv)
	_, errOut, code := h.run("add", "iPhone 13 128 Go")
	expect(t, code, ExitDenied, "", errOut)
	mustContain(t, errOut, "does not have the products:write scope", `"Read and add" access`)

	h.signedIn(fakeapi.WriteToken)
	h.srv.User.Limits.MaxProducts = 1
	h.srv.Products = []api.Product{{ID: "p1", Name: "Already tracked"}}
	_, errOut, code = h.run("add", "iPhone 13 128 Go")
	expect(t, code, ExitDenied, "", errOut)
	mustContain(t, errOut, "The Free plan tracks 1 product")
}

func TestUsageErrors(t *testing.T) {
	h := newHarness(t).signedIn(fakeapi.WriteToken)
	for _, args := range [][]string{
		{"add"},
		{"add", "x", "--condition", "both"},
		{"add", "x", "--max-price", "cheap"},
		{"add", "x", "--max-price", "0"},
		{"add", "x", "--type", "toaster"},
		{"add", "x", "--no-such-flag"},
		{"auth", "status", "extra"},
		{"no-such-command"},
	} {
		_, errOut, code := h.run(args...)
		if code != ExitUsage {
			t.Errorf("%v: exit code %d, want %d (%s)", args, code, ExitUsage, errOut)
		}
	}
}

func TestPrivateSiteRedirect(t *testing.T) {
	login := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://login.example/?next=/", http.StatusTemporaryRedirect)
	}))
	defer login.Close()
	h := newHarness(t).signedIn(fakeapi.WriteToken)
	h.env[config.EnvURL] = login.URL
	_, errOut, code := h.run("prices")
	expect(t, code, ExitFailure, "", errOut)
	mustContain(t, errOut, "redirected the request to https://login.example/?next=/", "the site is public")
}

func TestProxyInFrontRefuses(t *testing.T) {
	// exe.dev answers a private site's requests that carry credentials.
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "invalid or missing authentication", http.StatusUnauthorized)
	}))
	defer proxy.Close()
	h := newHarness(t).signedIn(fakeapi.WriteToken)
	h.env[config.EnvURL] = proxy.URL
	_, errOut, code := h.run("prices")
	expect(t, code, ExitFailure, "", errOut)
	mustContain(t, errOut, "refused the request before it reached Pricewatch", "private")
	if strings.Contains(errOut, "create a token") {
		t.Fatalf("suggests a new token for a proxy refusal: %s", errOut)
	}
}

func TestVersion(t *testing.T) {
	out, errOut, code := newHarness(t).run("version")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "pricewatch dev")
}

func TestParseEuros(t *testing.T) {
	for in, want := range map[string]int64{
		"450": 45000, "449,90": 44990, "449.9": 44990, "1 299,99 €": 129999, "12€": 1200, " 0,50 ": 50,
	} {
		if got, err := ParseEuros(in); err != nil || got != want {
			t.Errorf("ParseEuros(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "abc", "1,999", "-5", "0", "1.2.3", "€"} {
		if _, err := ParseEuros(in); err == nil {
			t.Errorf("ParseEuros(%q) accepted", in)
		}
	}
}

func TestDescribeFallsBackToFailure(t *testing.T) {
	if code, msg := describe(errors.New("connection refused")); code != ExitFailure || msg != "connection refused" {
		t.Fatalf("%d %q", code, msg)
	}
}
