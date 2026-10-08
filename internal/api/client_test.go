package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestsCarryTheTokenAndUserAgent(t *testing.T) {
	var got *http.Request
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = w.Write([]byte(`{"id":"p1","name":"Elden Ring (PS5)","condition":"any","targetPriceCents":4000,"watches":[]}`))
	}))
	defer srv.Close()
	target := int64(4000)
	p, err := New(srv.URL+"/", "pwt_x", "pricewatch-cli/test", nil).Track(context.Background(), "cat 1", TrackRequest{Condition: "any", TargetPriceCents: &target})
	if err != nil || p.Name != "Elden Ring (PS5)" || *p.TargetPriceCents != 4000 {
		t.Fatalf("%+v %v", p, err)
	}
	if got.Method != http.MethodPost || got.URL.EscapedPath() != "/api/v1/catalog/cat%201/track" {
		t.Fatalf("%s %s", got.Method, got.URL.EscapedPath())
	}
	if got.Header.Get(TokenHeader) != "pwt_x" || got.Header.Get("Authorization") != "" || got.Header.Get("User-Agent") != "pricewatch-cli/test" ||
		got.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("headers %v", got.Header)
	}
	if body["condition"] != "any" || body["targetPriceCents"] != 4000.0 {
		t.Fatalf("body %v", body)
	}
}

func TestErrors(t *testing.T) {
	cases := []struct {
		status      int
		body        string
		wantCode    string
		wantMessage string
	}{
		{403, `{"error":"this API token does not have the write scope","code":"insufficient_scope"}`,
			CodeInsufficientScope, "this API token does not have the write scope"},
		{500, `<html>oops</html>`, "", "the server answered 500 Internal Server Error"},
		{401, `invalid or missing authentication`, "", " refused the request before it reached Pricewatch (401 Unauthorized): the site may be private, or need a sign-in"},
	}
	for _, c := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(c.status)
			_, _ = w.Write([]byte(c.body))
		}))
		_, err := New(srv.URL, "pwt_x", "ua", nil).Products(context.Background())
		srv.Close()
		if StatusOf(err) != c.status || CodeOf(err) != c.wantCode || !strings.HasSuffix(err.Error(), c.wantMessage) ||
			FromPricewatch(err) != (c.wantCode != "") {
			t.Errorf("status %d: got %d %q %q", c.status, StatusOf(err), CodeOf(err), err)
		}
	}
}

func TestAnswerThatIsNotJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<!doctype html><title>Login</title>"))
	}))
	defer srv.Close()
	_, err := New(srv.URL, "pwt_x", "ua", nil).Me(context.Background())
	if err == nil || !strings.Contains(err.Error(), "did not answer with JSON") {
		t.Fatalf("%v", err)
	}
}

func TestSearchCatalogQuery(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	if _, err := New(srv.URL, "", "ua", nil).SearchCatalog(context.Background(), CatalogQuery{Text: "pokémon & co", Limit: 5}); err != nil {
		t.Fatal(err)
	}
	if query != "limit=5&q=pok%C3%A9mon+%26+co" {
		t.Fatal(query)
	}
	if _, err := New(srv.URL, "", "ua", nil).SearchCatalog(context.Background(), CatalogQuery{Type: "tcg", Store: "leclerc", Limit: 20}); err != nil {
		t.Fatal(err)
	}
	if query != "limit=20&q=&store=leclerc&type=tcg" {
		t.Fatal(query)
	}
}
