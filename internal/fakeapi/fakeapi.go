// Package fakeapi is an in-memory Pricewatch /api/v1 for tests. It follows
// the server's rules the CLI depends on: API tokens and their scopes,
// catalog search by words, discoveries, tracking, and the plan limit.
// Pricewatch's own tests cover the real server (backend/internal/api).
package fakeapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DharmaBytesX/pricewatch-cli/internal/api"
	"github.com/DharmaBytesX/pricewatch-cli/internal/match"
)

// Token is an API token the server accepts.
type Token struct {
	Name      string
	Scopes    []string
	ExpiresAt time.Time
}

// Tokens for tests.
var (
	WriteToken = "pwt_" + strings.Repeat("a", 48) // products:write
	ReadToken  = "pwt_" + strings.Repeat("b", 48) // products:read
)

// Server is the fake. Change its fields before the requests that use them.
type Server struct {
	*httptest.Server

	mu       sync.Mutex
	User     api.Me
	Tokens   map[string]Token
	Stores   []api.Store
	Catalog  []api.CatalogProduct
	Products []api.Product
	// Discover decides what a search of the stores finds: the product to
	// add to the catalog, or nil when no store sells it.
	Discover func(name string) *api.CatalogProduct
	// DiscoveryPolls is how many times a discovery answers "pending".
	DiscoveryPolls int
	// CheckAfterPolls: GET /products reports the first prices of new
	// watches from this call on (0: never).
	CheckAfterPolls int
	// Requests logs "METHOD /path body" for each request.
	Requests []string

	discoveries  map[string]*discovery
	productPolls int
	nextID       int
}

type discovery struct {
	name   string
	ptype  string
	polls  int
	result *api.CatalogProduct
}

// New starts a fake server with a Premium user and both test tokens.
func New() *Server {
	s := &Server{
		User: api.Me{ID: "user-1", Email: "lex@example.com", Plan: "premium",
			Limits: api.Plan{Name: "premium", CheckIntervalSeconds: 60}},
		Tokens: map[string]Token{
			WriteToken: {Name: "laptop", Scopes: []string{"products:write"}, ExpiresAt: time.Date(2027, 1, 6, 12, 0, 0, 0, time.UTC)},
			ReadToken:  {Name: "dashboard", Scopes: []string{"products:read"}, ExpiresAt: time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)},
		},
		Stores: []api.Store{
			{Name: "amazon", DisplayName: "Amazon FR"},
			{Name: "cdiscount", DisplayName: "Cdiscount"},
			{Name: "micromania", DisplayName: "Micromania"},
		},
		discoveries: map[string]*discovery{},
	}
	s.Server = httptest.NewServer(http.HandlerFunc(s.serve))
	return s
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	raw, _ := json.Marshal(body)
	s.Requests = append(s.Requests, strings.TrimSpace(r.Method+" "+r.URL.RequestURI()+" "+strings.TrimPrefix(string(raw), "null")))

	// Like Pricewatch: the token header, or Authorization: Bearer.
	credential := r.Header.Get(api.TokenHeader)
	if credential == "" {
		credential = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	token, ok := s.Tokens[credential]
	if !ok {
		reply(w, http.StatusUnauthorized, map[string]string{"error": "invalid, expired or revoked API token"})
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	route := r.Method + " " + path
	switch {
	case route == "GET /me":
		me := s.User
		me.Token = &api.TokenInfo{Name: token.Name, Scopes: token.Scopes, ExpiresAt: token.ExpiresAt}
		reply(w, http.StatusOK, me)
	case route == "GET /stores":
		reply(w, http.StatusOK, s.Stores)
	case !s.allowed(w, token, r.Method):
	case route == "GET /products":
		s.productPolls++
		if s.CheckAfterPolls > 0 && s.productPolls >= s.CheckAfterPolls {
			s.checkAll()
		}
		reply(w, http.StatusOK, s.Products)
	case route == "GET /catalog":
		s.searchCatalog(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(path, "/catalog/") && strings.HasSuffix(path, "/track"):
		s.track(w, strings.TrimSuffix(strings.TrimPrefix(path, "/catalog/"), "/track"), body)
	case route == "POST /discoveries":
		s.startDiscovery(w, body)
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/products/"):
		s.remove(w, strings.TrimPrefix(path, "/products/"))
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/discoveries/"):
		s.discovery(w, strings.TrimPrefix(path, "/discoveries/"))
	default:
		reply(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

// allowed applies the scopes: reading needs products:read or
// products:write, changing needs products:write.
func (s *Server) allowed(w http.ResponseWriter, t Token, method string) bool {
	has := func(scope string) bool {
		for _, sc := range t.Scopes {
			if sc == scope {
				return true
			}
		}
		return false
	}
	need := "products:read"
	if method != http.MethodGet {
		need = "products:write"
	}
	if has(need) || (need == "products:read" && has("products:write")) {
		return true
	}
	reply(w, http.StatusForbidden, map[string]string{
		"error": "this API token does not have the " + need + " scope", "code": "insufficient_scope"})
	return false
}

func (s *Server) searchCatalog(w http.ResponseWriter, r *http.Request) {
	q, ptype := r.URL.Query().Get("q"), r.URL.Query().Get("type")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out := []api.CatalogProduct{}
	for _, c := range s.Catalog {
		if ptype != "" && c.Type != ptype {
			continue
		}
		if len(c.Offers) > 0 && match.AllWords(c.Name, q) && (limit == 0 || len(out) < limit) {
			out = append(out, c)
		}
	}
	reply(w, http.StatusOK, out)
}

func (s *Server) track(w http.ResponseWriter, catalogID string, body map[string]any) {
	var c *api.CatalogProduct
	for i := range s.Catalog {
		if s.Catalog[i].ID == catalogID {
			c = &s.Catalog[i]
		}
	}
	if c == nil {
		reply(w, http.StatusNotFound, map[string]string{"error": "catalog product not found"})
		return
	}
	if limit := s.User.Limits.MaxProducts; limit > 0 && len(s.Products) >= limit {
		reply(w, http.StatusForbidden, map[string]string{
			"error": fmt.Sprintf("The Free plan tracks %d product. Premium and Ultimate track as many as you want.", limit),
			"code":  "plan_limit"})
		return
	}
	p := api.Product{ID: s.id("product"), Name: c.Name, Type: c.Type, CreatedAt: time.Now()}
	p.Condition, _ = body["condition"].(string)
	if v, ok := body["targetPriceCents"].(float64); ok {
		cents := int64(v)
		p.TargetPriceCents = &cents
	}
	for _, o := range c.Offers {
		p.Watches = append(p.Watches, api.Watch{ID: s.id("watch"), Marketplace: o.Marketplace, URL: o.URL, Active: true})
	}
	s.Products = append([]api.Product{p}, s.Products...)
	reply(w, http.StatusCreated, p)
}

func (s *Server) startDiscovery(w http.ResponseWriter, body map[string]any) {
	name, _ := body["name"].(string)
	ptype, _ := body["type"].(string)
	for i := range s.Catalog {
		if match.Same(s.Catalog[i].Name, name) && len(s.Catalog[i].Offers) > 0 {
			reply(w, http.StatusOK, map[string]any{"status": "exists", "catalog": s.Catalog[i]})
			return
		}
	}
	d := &discovery{name: name, ptype: ptype}
	if s.Discover != nil {
		d.result = s.Discover(name)
	}
	id := s.id("discovery")
	s.discoveries[id] = d
	reply(w, http.StatusAccepted, map[string]string{"status": "requested", "discoveryId": id})
}

func (s *Server) discovery(w http.ResponseWriter, id string) {
	d, ok := s.discoveries[id]
	if !ok {
		reply(w, http.StatusNotFound, map[string]string{"error": "discovery not found"})
		return
	}
	out := api.Discovery{ID: id, Name: d.name, Status: api.DiscoveryPending}
	d.polls++
	if d.polls > s.DiscoveryPolls {
		if d.result == nil {
			out.Status = api.DiscoveryEmpty
		} else {
			if !s.inCatalog(d.result.ID) {
				s.Catalog = append(s.Catalog, *d.result)
			}
			out.Status, out.Catalog = api.DiscoveryPublished, d.result
			out.Result = &api.DiscoveryResult{}
			for _, o := range d.result.Offers {
				out.Result.Offers = append(out.Result.Offers, api.DiscoveryOffer{Marketplace: o.Marketplace, URL: o.URL})
			}
		}
	}
	reply(w, http.StatusOK, out)
}

func (s *Server) remove(w http.ResponseWriter, id string) {
	for i, p := range s.Products {
		if p.ID == id {
			s.Products = append(s.Products[:i], s.Products[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	reply(w, http.StatusNotFound, map[string]string{"error": "product not found"})
}

// DiscoveryType returns the type the last discovery of name asked for.
func (s *Server) DiscoveryType(name string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.discoveries {
		if d.name == name {
			return d.ptype
		}
	}
	return ""
}

// Log returns the requests received so far.
func (s *Server) Log() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.Requests...)
}

func (s *Server) inCatalog(id string) bool {
	for _, c := range s.Catalog {
		if c.ID == id {
			return true
		}
	}
	return false
}

// checkAll gives every unchecked watch a first price: 100 €, 110 €, …
func (s *Server) checkAll() {
	now := time.Now()
	for i := range s.Products {
		for j := range s.Products[i].Watches {
			w := &s.Products[i].Watches[j]
			if w.LastCheckedAt == nil {
				price := int64(10000 + 1000*j)
				w.LastCheckedAt = &now
				w.Latest = &api.Snapshot{PriceCents: &price, InStock: j%2 == 0, CapturedAt: now}
			}
		}
	}
}

func (s *Server) id(kind string) string {
	s.nextID++
	return fmt.Sprintf("%s-%08d", kind, s.nextID)
}

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
