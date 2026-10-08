// Package fakeapi is an in-memory Pricewatch /api/v1 for tests. It follows
// the server's rules the CLI depends on: API tokens and their scopes,
// catalog search by words, discoveries, tracking, the plan limit, and the
// webhook.
// Pricewatch's own tests cover the real server (backend/internal/api).
package fakeapi

import (
	"encoding/base64"
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
	WriteToken   = "pwt_" + strings.Repeat("a", 48) // write
	ReadToken    = "pwt_" + strings.Repeat("b", 48) // read
	WebhookToken = "pwt_" + strings.Repeat("c", 48) // write
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

	// WebhookURL is the user's webhook; nil when none is set.
	WebhookURL *string
	// WebhookSecret is the webhook's signing secret.
	WebhookSecret string
	// Deliveries are the webhook's deliveries, newest first.
	Deliveries []api.WebhookDelivery
	// TestStatus is how the webhook answers a test event: an HTTP status
	// (0: 200), or -1 for no answer (TestError says why).
	TestStatus int
	TestError  string

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
			WriteToken: {Name: "laptop", Scopes: []string{"write"}, ExpiresAt: time.Date(2027, 1, 6, 12, 0, 0, 0, time.UTC)},
			ReadToken:  {Name: "dashboard", Scopes: []string{"read"}, ExpiresAt: time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)},
			WebhookToken: {Name: "automation", Scopes: []string{"write"},
				ExpiresAt: time.Date(2027, 1, 6, 12, 0, 0, 0, time.UTC)},
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
	case strings.HasPrefix(path, "/webhook"):
		s.webhook(w, r, route, body)
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

// allowed applies the scopes: reading needs read or write, changing
// needs write.
func (s *Server) allowed(w http.ResponseWriter, t Token, method string) bool {
	need := "read"
	if method != http.MethodGet {
		need = "write"
	}
	if hasScope(t, need) || (need == "read" && hasScope(t, "write")) {
		return true
	}
	denyScope(w, need)
	return false
}

func hasScope(t Token, scope string) bool {
	for _, sc := range t.Scopes {
		if sc == scope {
			return true
		}
	}
	return false
}

func denyScope(w http.ResponseWriter, scope string) {
	reply(w, http.StatusForbidden, map[string]string{
		"error": "this API token does not have the " + scope + " scope", "code": "insufficient_scope"})
}

// webhook serves /webhook and its routes.
func (s *Server) webhook(w http.ResponseWriter, r *http.Request, route string, body map[string]any) {
	noWebhook := map[string]string{"error": "no webhook is set"}
	switch route {
	case "GET /webhook":
		reply(w, http.StatusOK, api.Webhook{URL: s.WebhookURL})
	case "PUT /webhook":
		address, _ := body["url"].(string)
		if !strings.HasPrefix(address, "https://") {
			reply(w, http.StatusBadRequest, map[string]string{"error": "the webhook address must start with https://"})
			return
		}
		s.WebhookURL = &address
		out := api.Webhook{URL: s.WebhookURL}
		if s.WebhookSecret == "" {
			s.WebhookSecret = s.secret()
			out.Secret = s.WebhookSecret
		}
		reply(w, http.StatusOK, out)
	case "DELETE /webhook":
		s.WebhookURL, s.WebhookSecret = nil, ""
		reply(w, http.StatusOK, map[string]bool{"removed": true})
	case "POST /webhook/secret":
		if s.WebhookURL == nil {
			reply(w, http.StatusNotFound, noWebhook)
			return
		}
		s.WebhookSecret = s.secret()
		reply(w, http.StatusOK, map[string]string{"secret": s.WebhookSecret})
	case "POST /webhook/test":
		if s.WebhookURL == nil {
			reply(w, http.StatusNotFound, noWebhook)
			return
		}
		s.nextID++
		created := time.Now().UTC().Truncate(time.Second)
		d := api.WebhookDelivery{ID: int64(s.nextID), EventID: fmt.Sprintf("msg_test%08d", s.nextID), Type: "test",
			Status: api.DeliverySent, Attempts: 1, CreatedAt: created}
		switch status := s.TestStatus; {
		case status < 0:
			msg := s.TestError
			d.Status, d.Error = api.DeliveryFailed, &msg
		case status >= 200 && status < 300 || status == 0:
			if status == 0 {
				status = http.StatusOK
			}
			delivered := created.Add(120 * time.Millisecond)
			d.ResponseStatus, d.DeliveredAt = &status, &delivered
		default:
			msg := fmt.Sprintf("the webhook answered %d", status)
			d.Status, d.ResponseStatus, d.Error = api.DeliveryFailed, &status, &msg
		}
		s.Deliveries = append([]api.WebhookDelivery{d}, s.Deliveries...)
		reply(w, http.StatusOK, map[string]any{"delivery": d})
	case "GET /webhook/deliveries":
		limit := 20
		if v := r.URL.Query().Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 || n > 100 {
				reply(w, http.StatusBadRequest, map[string]string{"error": "limit must be between 1 and 100"})
				return
			}
			limit = n
		}
		out := append([]api.WebhookDelivery{}, s.Deliveries...)
		if len(out) > limit {
			out = out[:limit]
		}
		reply(w, http.StatusOK, map[string]any{"deliveries": out})
	default:
		reply(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

// secret returns a new signing secret: whsec_ and 32 bytes in base64.
func (s *Server) secret() string {
	s.nextID++
	return "whsec_" + base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%032d", s.nextID)))
}

func (s *Server) searchCatalog(w http.ResponseWriter, r *http.Request) {
	q, ptype, store := r.URL.Query().Get("q"), r.URL.Query().Get("type"), r.URL.Query().Get("store")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out := []api.CatalogProduct{}
	for _, c := range s.Catalog {
		if ptype != "" && c.Type != ptype || store != "" && !hasOffer(c, store) {
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

func hasOffer(c api.CatalogProduct, store string) bool {
	for _, o := range c.Offers {
		if o.Marketplace == store {
			return true
		}
	}
	return false
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
