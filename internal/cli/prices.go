package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/DharmaBytesX/pricewatch-cli/internal/api"
	"github.com/DharmaBytesX/pricewatch-cli/internal/match"
	"github.com/DharmaBytesX/pricewatch-cli/internal/render"
)

func newPricesCommand(app *App, g *globals) *cobra.Command {
	var asJSON, inStock bool
	cmd := &cobra.Command{
		Use:   "prices [PRODUCT]",
		Short: "Show the prices and stock of your tracked products",
		Long: `Without PRODUCT, lists your tracked products with their lowest price in stock.

With PRODUCT, shows each store's price, stock, last check and link for the
tracked products whose name has every word of PRODUCT (case and accents
ignored), or whose ID starts with PRODUCT.

Prices are the latest the stores showed: your plan sets how often they are
checked.`,
		Example: `  pricewatch prices
  pricewatch prices "iphone 13 128"
  pricewatch prices elden ring --in-stock
  pricewatch prices --json | jq '.[].name'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := app.client(g)
			if err != nil {
				return err
			}
			products, err := client.Products(cmd.Context())
			if err != nil {
				return err
			}
			query := strings.TrimSpace(strings.Join(args, " "))
			if query != "" {
				products = findProducts(products, query)
				if len(products) == 0 {
					return &exitError{code: ExitNotFound, msg: fmt.Sprintf(
						"you track no product matching “%s”; to track it: pricewatch add \"%s\"", query, query)}
				}
			}
			if asJSON {
				return app.writeJSON(products)
			}
			if len(products) == 0 {
				fmt.Fprintln(app.Out, `You track no product yet. Add one with: pricewatch add "PRODUCT NAME"`)
				return nil
			}
			names := app.storeNames(cmd.Context(), client)
			if query == "" {
				return writeSummary(app.Out, products, names, app.Now())
			}
			for i, p := range products {
				if i > 0 {
					fmt.Fprintln(app.Out)
				}
				if err := writeDetail(app.Out, p, names, app.Now(), inStock); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the products as JSON, as the API returns them")
	cmd.Flags().BoolVar(&inStock, "in-stock", false, "show only the stores that have the product in stock")
	return cmd
}

// findProducts returns the products whose name has every word of query, or
// whose ID starts with it (8 characters at least).
func findProducts(products []api.Product, query string) []api.Product {
	var out []api.Product
	for _, p := range products {
		byID := len(query) >= 8 && strings.HasPrefix(p.ID, strings.ToLower(query))
		if byID || match.AllWords(p.Name, query) {
			out = append(out, p)
		}
	}
	return out
}

// storeNames maps store code names to display names; when the list cannot
// be read, the code names are shown.
func (app *App) storeNames(ctx context.Context, client *api.Client) func(string) string {
	names := map[string]string{}
	if stores, err := client.Stores(ctx); err == nil {
		for _, s := range stores {
			names[s.Name] = s.DisplayName
		}
	}
	return func(code string) string {
		if n := names[code]; n != "" {
			return n
		}
		return code
	}
}

// best returns the watch with the lowest price, preferring stores that
// have the product in stock; nil when no store shows a price.
func best(p api.Product) *api.Watch {
	var b *api.Watch
	for i := range p.Watches {
		w := &p.Watches[i]
		if !w.Active || w.Latest == nil || w.Latest.PriceCents == nil {
			continue
		}
		if b == nil || less(w, b) {
			b = w
		}
	}
	return b
}

// less orders store rows: in stock first, then by price, then the stores
// without a price, then paused ones.
func less(a, b *api.Watch) bool {
	rank := func(w *api.Watch) int {
		switch {
		case !w.Active:
			return 3
		case w.Latest == nil || w.Latest.PriceCents == nil:
			return 2
		case !w.Latest.InStock:
			return 1
		}
		return 0
	}
	if ra, rb := rank(a), rank(b); ra != rb {
		return ra < rb
	}
	if rank(a) <= 1 {
		return *a.Latest.PriceCents < *b.Latest.PriceCents
	}
	return a.Marketplace < b.Marketplace
}

func stockText(w api.Watch) string {
	switch {
	case !w.Active:
		return "paused"
	case w.Latest != nil && w.Latest.InStock:
		return "in stock"
	case w.Latest != nil:
		return "out of stock"
	case w.LastCheckedAt == nil:
		return "checking…"
	}
	return "no data"
}

func priceOf(w api.Watch) *int64 {
	if w.Latest == nil {
		return nil
	}
	return w.Latest.PriceCents
}

// waitingForFirstCheck reports whether a store of the product is active and
// was never checked: its first check comes a few seconds after tracking.
func waitingForFirstCheck(p api.Product) bool {
	for _, w := range p.Watches {
		if w.Active && w.LastCheckedAt == nil {
			return true
		}
	}
	return false
}

func lastChecked(p api.Product) *time.Time {
	var last *time.Time
	for _, w := range p.Watches {
		if w.LastCheckedAt != nil && (last == nil || w.LastCheckedAt.After(*last)) {
			last = w.LastCheckedAt
		}
	}
	return last
}

func writeSummary(out io.Writer, products []api.Product, names func(string) string, now time.Time) error {
	t := render.NewTable(out, "PRODUCT", "LOWEST", "STORE", "IN STOCK", "CHECKED")
	for _, p := range products {
		lowest, store := "—", "—"
		if b := best(p); b != nil {
			lowest, store = render.Euros(*b.Latest.PriceCents), names(b.Marketplace)
			if !b.Latest.InStock {
				lowest += " (out of stock)"
			}
		}
		in := 0
		for _, w := range p.Watches {
			if w.Active && w.Latest != nil && w.Latest.InStock {
				in++
			}
		}
		checked := render.Ago(lastChecked(p), now)
		if lastChecked(p) == nil && waitingForFirstCheck(p) {
			checked = "checking…"
		}
		t.Row(p.Name, lowest, store, fmt.Sprintf("%d of %d stores", in, len(p.Watches)), checked)
	}
	return t.Flush()
}

func writeDetail(out io.Writer, p api.Product, names func(string) string, now time.Time, inStockOnly bool) error {
	fmt.Fprintln(out, p.Name)
	facts := []string{render.Condition(p.Condition)}
	if p.TargetPriceCents != nil {
		facts = append(facts, "alerts at or below "+render.Euros(*p.TargetPriceCents))
	}
	if b := best(p); b != nil && b.Latest.InStock {
		facts = append(facts, fmt.Sprintf("lowest in stock %s at %s", render.Euros(*b.Latest.PriceCents), names(b.Marketplace)))
	} else {
		facts = append(facts, "no store has it in stock")
	}
	fmt.Fprintln(out, strings.Join(facts, " · "))
	fmt.Fprintln(out)

	var watches []api.Watch
	for _, w := range p.Watches {
		if !inStockOnly || (w.Active && w.Latest != nil && w.Latest.InStock) {
			watches = append(watches, w)
		}
	}
	if len(watches) == 0 {
		_, err := fmt.Fprintln(out, "No store to show.")
		return err
	}
	sort.SliceStable(watches, func(i, j int) bool { return less(&watches[i], &watches[j]) })
	t := render.NewTable(out, "STORE", "PRICE", "STOCK", "CHECKED", "LINK")
	for _, w := range watches {
		t.Row(names(w.Marketplace), render.OptionalEuros(priceOf(w)), stockText(w), render.Ago(w.LastCheckedAt, now), w.URL)
	}
	return t.Flush()
}
