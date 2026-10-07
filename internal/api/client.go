// Package api is a client for Pricewatch's /api/v1, the API of the CLI.
//
// Every request carries the user's API token. Errors from the server are
// returned as *Error, with the HTTP status and the server's error code.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultTimeout bounds each request.
const DefaultTimeout = 30 * time.Second

// TokenHeader carries the API token. Pricewatch also reads Authorization:
// Bearer, but a proxy in front of the site may answer requests with an
// Authorization header itself (exe.dev does for a private site).
const TokenHeader = "X-Pricewatch-Token"

// Client calls one Pricewatch server with one API token.
type Client struct {
	baseURL   string
	token     string
	userAgent string
	http      *http.Client
}

// New returns a client for the server at baseURL (e.g.
// https://pricewatch.exe.xyz). httpClient may be nil.
func New(baseURL, token, userAgent string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultTimeout}
	}
	// Do not follow redirects: a private site answers with a redirect to
	// its login page, which must be reported, not parsed.
	c := *httpClient
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), token: token, userAgent: userAgent, http: &c}
}

// Me returns the user and the token the client uses.
func (c *Client) Me(ctx context.Context) (*Me, error) {
	var out Me
	return &out, c.do(ctx, http.MethodGet, "/me", nil, &out)
}

// Stores lists the stores Pricewatch checks.
func (c *Client) Stores(ctx context.Context) ([]Store, error) {
	var out []Store
	return out, c.do(ctx, http.MethodGet, "/stores", nil, &out)
}

// Products lists the user's tracked products, newest first, with each
// store's latest price and stock.
func (c *Client) Products(ctx context.Context) ([]Product, error) {
	var out []Product
	return out, c.do(ctx, http.MethodGet, "/products", nil, &out)
}

// SearchCatalog lists the catalog products whose name has every word of
// query, best match first. ptype keeps one product type; "" any.
func (c *Client) SearchCatalog(ctx context.Context, query, ptype string, limit int) ([]CatalogProduct, error) {
	q := url.Values{"q": {query}, "limit": {strconv.Itoa(limit)}}
	if ptype != "" {
		q.Set("type", ptype)
	}
	var out []CatalogProduct
	return out, c.do(ctx, http.MethodGet, "/catalog?"+q.Encode(), nil, &out)
}

// Track starts tracking a catalog product and returns the tracked product.
func (c *Client) Track(ctx context.Context, catalogID string, req TrackRequest) (*Product, error) {
	var out Product
	return &out, c.do(ctx, http.MethodPost, "/catalog/"+url.PathEscape(catalogID)+"/track", req, &out)
}

// StartDiscovery asks for a product to be found in every store. ptype may
// be empty: the server guesses it from the name.
func (c *Client) StartDiscovery(ctx context.Context, name, ptype string) (*DiscoveryStart, error) {
	var out DiscoveryStart
	body := map[string]string{"name": name, "type": ptype}
	return &out, c.do(ctx, http.MethodPost, "/discoveries", body, &out)
}

// Discovery returns a discovery's progress or result.
func (c *Client) Discovery(ctx context.Context, id string) (*Discovery, error) {
	var out Discovery
	return &out, c.do(ctx, http.MethodGet, "/discoveries/"+url.PathEscape(id), nil, &out)
}

// do sends a request to /api/v1 and decodes the JSON answer into out.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+"/api/v1"+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set(TokenHeader, c.token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach %s: %w", c.baseURL, err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 10<<20))
	if err != nil {
		return fmt.Errorf("read the answer of %s: %w", c.baseURL, err)
	}
	if res.StatusCode >= 300 {
		return newError(c.baseURL, res, data)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return &Error{StatusCode: res.StatusCode, Message: fmt.Sprintf("%s did not answer with JSON; is it the Pricewatch address?", c.baseURL)}
	}
	return nil
}

// Error is an error answer from the server.
type Error struct {
	StatusCode int
	Code       string // e.g. "insufficient_scope", "plan_limit"; may be empty
	Message    string
	// FromPricewatch is false when something in front of Pricewatch (a
	// proxy, a sign-in page) answered.
	FromPricewatch bool
}

func (e *Error) Error() string { return e.Message }

// Error codes the server sends.
const (
	CodeInsufficientScope = "insufficient_scope"
	CodePlanLimit         = "plan_limit"
)

func newError(baseURL string, res *http.Response, data []byte) *Error {
	e := &Error{StatusCode: res.StatusCode}
	if res.StatusCode >= 300 && res.StatusCode < 400 {
		e.Message = fmt.Sprintf("the server redirected the request to %s: check the Pricewatch address, and that the site is public",
			res.Header.Get("Location"))
		return e
	}
	var body struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	switch {
	case json.Unmarshal(data, &body) == nil && body.Error != "":
		e.Message, e.Code, e.FromPricewatch = body.Error, body.Code, true
	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden:
		e.Message = fmt.Sprintf("%s refused the request before it reached Pricewatch (%s): the site may be private, or need a sign-in",
			baseURL, res.Status)
	default:
		e.Message = fmt.Sprintf("the server answered %s", res.Status)
	}
	return e
}

// FromPricewatch reports whether err is an error answer from Pricewatch
// itself.
func FromPricewatch(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.FromPricewatch
}

// StatusOf returns the HTTP status of an *Error in err's chain, or 0.
func StatusOf(err error) int {
	var e *Error
	if errors.As(err, &e) {
		return e.StatusCode
	}
	return 0
}

// CodeOf returns the server's error code of an *Error in err's chain.
func CodeOf(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
