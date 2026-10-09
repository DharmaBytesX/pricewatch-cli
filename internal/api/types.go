package api

import "time"

// The types below follow Pricewatch's /api/v1 responses. The API only adds
// fields over time, so unknown fields are ignored.

// Me is the authenticated user.
type Me struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Plan   string `json:"plan"`
	Limits Plan   `json:"limits"`
	// Token is the API token used; nil for a website session.
	Token *TokenInfo `json:"token,omitempty"`
}

// Plan is what the user's plan allows.
type Plan struct {
	Name                 string `json:"name"`
	MaxProducts          int    `json:"maxProducts"` // 0: no limit
	CheckIntervalSeconds int    `json:"checkIntervalSeconds"`
}

// TokenInfo describes an API token.
type TokenInfo struct {
	Name      string    `json:"name"`
	Scopes    []string  `json:"scopes"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Store is a store Pricewatch checks.
type Store struct {
	Name        string `json:"name"`        // code name, e.g. "cdiscount"
	DisplayName string `json:"displayName"` // e.g. "Cdiscount"
}

// Product is a product the user tracks.
type Product struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	Condition        string    `json:"condition"` // new, used or any
	TargetPriceCents *int64    `json:"targetPriceCents"`
	CreatedAt        time.Time `json:"createdAt"`
	Watches          []Watch   `json:"watches"`
}

// Watch is one store's page for a tracked product.
type Watch struct {
	ID            string     `json:"id"`
	Marketplace   string     `json:"marketplace"` // store code name
	URL           string     `json:"url"`
	Active        bool       `json:"active"`
	LastCheckedAt *time.Time `json:"lastCheckedAt"`
	LastStatus    *string    `json:"lastStatus"`
	Latest        *Snapshot  `json:"latestSnapshot"`
	// LinkOnly: Pricewatch never reads this store (Amazon): the watch is
	// a link, with no price, stock or check.
	LinkOnly bool `json:"linkOnly,omitempty"`
}

// Snapshot is the latest price and stock read on a store's page.
type Snapshot struct {
	PriceCents *int64    `json:"priceCents"`
	Currency   *string   `json:"currency"`
	InStock    bool      `json:"inStock"`
	Title      *string   `json:"title"`
	CapturedAt time.Time `json:"capturedAt"`
}

// CatalogProduct is a product of the shared catalog, with the stores that
// sell it.
type CatalogProduct struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Offers []CatalogOffer `json:"offers"`
}

// CatalogOffer is one store's page for a catalog product.
type CatalogOffer struct {
	ID          string `json:"id"`
	Marketplace string `json:"marketplace"`
	URL         string `json:"url"`
}

// TrackRequest says how to track a catalog product.
type TrackRequest struct {
	Condition        string `json:"condition"` // new, used or any
	TargetPriceCents *int64 `json:"targetPriceCents,omitempty"`
}

// DiscoveryStart is the answer to a request for a new product: the catalog
// product when it already exists (Status "exists"), or the discovery that
// searches the stores for it.
type DiscoveryStart struct {
	Status      string          `json:"status"`
	DiscoveryID string          `json:"discoveryId"`
	Catalog     *CatalogProduct `json:"catalog"`
}

// Discovery statuses.
const (
	DiscoveryPending   = "pending"   // the stores are being searched
	DiscoveryPublished = "published" // found, and added to the catalog (Catalog is set)
	DiscoveryEmpty     = "empty"     // no store sells it
	DiscoveryDone      = "done"      // found, but not added to the catalog
)

// Discovery is a search of every store for a new product.
type Discovery struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Status  string           `json:"status"`
	Result  *DiscoveryResult `json:"result"`
	Catalog *CatalogProduct  `json:"catalog"`
}

// DiscoveryResult is what the stores answered so far.
type DiscoveryResult struct {
	Offers []DiscoveryOffer `json:"offers"`
	Stores []DiscoveryStore `json:"stores"`
}

// DiscoveryStore is one store's answer.
type DiscoveryStore struct {
	Store  string          `json:"store"`
	Millis int64           `json:"ms"`
	Error  string          `json:"error"`
	Offer  *DiscoveryOffer `json:"offer"`
}

// DiscoveryOffer is a store's page for the product.
type DiscoveryOffer struct {
	Marketplace string   `json:"marketplace"`
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Price       *float64 `json:"price"` // euros
	InStock     *bool    `json:"inStock"`
}

// Webhook is the address Pricewatch sends the user's alerts to.
type Webhook struct {
	URL *string `json:"url"` // nil when no webhook is set
	// Secret signs the requests. The server sends it only when it creates
	// it: for a new webhook, or a new secret.
	Secret string `json:"secret,omitempty"`
}

// Webhook delivery statuses.
const (
	DeliveryPending = "pending" // queued, or waiting for a retry
	DeliverySent    = "sent"
	DeliveryFailed  = "failed"
)

// WebhookDelivery is one event sent, or being sent, to the webhook.
type WebhookDelivery struct {
	ID       int64  `json:"id"`
	EventID  string `json:"eventId"` // the webhook-id header; the same on every retry
	Type     string `json:"type"`    // price_drop, restock or test
	AlertID  *int64 `json:"alertId"` // nil for a test event
	Status   string `json:"status"`
	Attempts int    `json:"attempts"`
	// ResponseStatus is the HTTP status of the last answer; nil when none
	// came (network error, timeout).
	ResponseStatus *int       `json:"responseStatus"`
	Error          *string    `json:"error"` // short reason; nil when sent
	CreatedAt      time.Time  `json:"createdAt"`
	DeliveredAt    *time.Time `json:"deliveredAt"`
}
