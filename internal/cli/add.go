package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/DharmaBytesX/pricewatch-cli/internal/api"
	"github.com/DharmaBytesX/pricewatch-cli/internal/match"
	"github.com/DharmaBytesX/pricewatch-cli/internal/render"
)

// Product types Pricewatch knows; "" lets the server guess from the name.
var productTypes = []string{"video_game", "console", "smartphone", "laptop", "tcg", "other"}

type addOptions struct {
	condition string
	maxPrice  string
	ptype     string
	id        string
	search    bool
	yes       bool
	wait      bool
	asJSON    bool
	timeout   time.Duration
}

func newAddCommand(app *App, g *globals) *cobra.Command {
	o := &addOptions{}
	cmd := &cobra.Command{
		Use:   "add NAME",
		Short: "Track a product: from the catalog, or found in every store",
		Long: `Tracks a product, the way the website's "Track a product" panel does.

pricewatch first looks for NAME in Pricewatch's catalog:
  - a product with exactly this name is tracked;
  - when several products match, it asks which one (or, without a terminal,
    lists them and stops: give the exact name, --id, or --yes);
  - when none matches, Pricewatch searches every store for NAME, adds what
    it finds to the catalog, and the product is tracked. This takes a few
    seconds.

The first prices arrive a few seconds after the product is tracked; --wait
shows them. Your plan limits how many products you track.`,
		Example: `  pricewatch add "Elden Ring PS5"
  pricewatch add "iPhone 13 128 Go" --condition any --max-price 450
  pricewatch add "Mario Kart World" --type video_game --wait
  pricewatch add --id 0b6f2c1e-1c4b-4f1e-9a51-7d3c2a1b9e00`,
		Args: func(_ *cobra.Command, args []string) error {
			if o.id == "" && len(args) == 0 {
				return usageError("give the product's NAME, or its catalog ID with --id")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.add(cmd.Context(), g, o, strings.TrimSpace(strings.Join(args, " ")))
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.condition, "condition", "new", "new, used, or any (used & new)")
	f.StringVar(&o.maxPrice, "max-price", "", "alert only at or below this price in euros, e.g. 450 or 449,90")
	f.StringVar(&o.ptype, "type", "", "type of a product the stores are searched for: "+strings.Join(productTypes, ", ")+" (default: guessed from NAME)")
	f.StringVar(&o.id, "id", "", "track this catalog product ID instead of searching by NAME")
	f.BoolVar(&o.search, "new", false, "search the stores for NAME even when the catalog has products that match")
	f.BoolVar(&o.yes, "yes", false, "when several catalog products match, take the best match without asking")
	f.BoolVar(&o.wait, "wait", false, "wait for the first prices and show them")
	f.BoolVar(&o.asJSON, "json", false, "print the tracked product as JSON")
	f.DurationVar(&o.timeout, "timeout", 2*time.Minute, "how long to wait for the store search and for --wait")
	return cmd
}

func (app *App) add(ctx context.Context, g *globals, o *addOptions, name string) error {
	req, err := o.trackRequest()
	if err != nil {
		return err
	}
	if o.ptype != "" && !contains(productTypes, o.ptype) {
		return usageError("--type must be one of: " + strings.Join(productTypes, ", "))
	}
	client, _, err := app.client(g)
	if err != nil {
		return err
	}

	catalogID := o.id
	if catalogID == "" {
		c, err := app.findInCatalog(ctx, client, o, name)
		if err != nil {
			return err
		}
		if c == nil {
			if c, err = app.findInStores(ctx, client, o, name); err != nil {
				return err
			}
		}
		catalogID = c.ID
	}

	product, err := client.Track(ctx, catalogID, req)
	if err != nil {
		return err
	}
	if o.wait {
		if product, err = app.waitForPrices(ctx, client, product.ID, o.timeout); err != nil {
			return err
		}
	}
	if o.asJSON {
		return app.writeJSON(product)
	}
	limits := render.Condition(product.Condition)
	if product.TargetPriceCents != nil {
		limits += ", alerts at or below " + render.Euros(*product.TargetPriceCents)
	}
	fmt.Fprintf(app.Out, "Tracking %s in %d stores (%s).\n", product.Name, len(product.Watches), limits)
	if !o.wait {
		fmt.Fprintf(app.Out, "The first prices arrive in a few seconds: pricewatch prices \"%s\"\n", product.Name)
		return nil
	}
	fmt.Fprintln(app.Out)
	return writeDetail(app.Out, *product, app.storeNames(ctx, client), app.Now(), false)
}

// trackRequest checks the condition and the maximum price.
func (o *addOptions) trackRequest() (api.TrackRequest, error) {
	req := api.TrackRequest{Condition: o.condition}
	if !contains([]string{"new", "used", "any"}, o.condition) {
		return req, usageError("--condition must be new, used or any")
	}
	if o.maxPrice != "" {
		cents, err := ParseEuros(o.maxPrice)
		if err != nil {
			return req, usageError("--max-price: " + err.Error())
		}
		req.TargetPriceCents = &cents
	}
	return req, nil
}

// findInCatalog returns the catalog product to track, or nil to search the
// stores.
func (app *App) findInCatalog(ctx context.Context, client *api.Client, o *addOptions, name string) (*api.CatalogProduct, error) {
	if o.search {
		return nil, nil
	}
	found, err := client.SearchCatalog(ctx, name, "", 10)
	if err != nil {
		return nil, err
	}
	for i := range found {
		if match.Same(found[i].Name, name) {
			return &found[i], nil
		}
	}
	switch {
	case len(found) == 0:
		return nil, nil
	case o.yes:
		return &found[0], nil
	case !app.StdinIsTerminal():
		var b strings.Builder
		fmt.Fprintf(&b, "“%s” matches %d catalog products:\n", name, len(found))
		for _, c := range found {
			fmt.Fprintf(&b, "  %s  (--id %s)\n", c.Name, c.ID)
		}
		fmt.Fprintf(&b, "Give the exact name, --id ID, --yes to take the first, or --new to search the stores for “%s”", name)
		return nil, usageError(b.String())
	}
	return app.choose(found, name)
}

// choose asks which catalog product to track; nil means "search the stores".
func (app *App) choose(found []api.CatalogProduct, name string) (*api.CatalogProduct, error) {
	fmt.Fprintf(app.Err, "“%s” matches these products:\n", name)
	for i, c := range found {
		fmt.Fprintf(app.Err, "  %d) %s (%d stores)\n", i+1, c.Name, len(c.Offers))
	}
	fmt.Fprintf(app.Err, "  0) none of these: search the stores for “%s”\n", name)
	for {
		fmt.Fprint(app.Err, "Choose [1]: ")
		line, err := app.readLine()
		if err != nil {
			return nil, usageError("no choice was made")
		}
		if line == "" {
			return &found[0], nil
		}
		n, err := strconv.Atoi(line)
		switch {
		case err == nil && n == 0:
			return nil, nil
		case err == nil && n >= 1 && n <= len(found):
			return &found[n-1], nil
		}
		fmt.Fprintf(app.Err, "Type a number from 0 to %d.\n", len(found))
	}
}

// findInStores asks Pricewatch to search every store for name, and waits
// until the product is in the catalog.
func (app *App) findInStores(ctx context.Context, client *api.Client, o *addOptions, name string) (*api.CatalogProduct, error) {
	start, err := client.StartDiscovery(ctx, name, o.ptype)
	if err != nil {
		return nil, err
	}
	if start.Catalog != nil {
		return start.Catalog, nil
	}
	if start.DiscoveryID == "" {
		return nil, fmt.Errorf("the server accepted the request but started no search; try again in a moment")
	}
	fmt.Fprintf(app.Err, "Searching the stores for “%s”…\n", name)
	deadline := app.Now().Add(o.timeout)
	for {
		d, err := client.Discovery(ctx, start.DiscoveryID)
		if err != nil {
			return nil, err
		}
		switch d.Status {
		case api.DiscoveryPublished:
			if d.Catalog != nil {
				if d.Result != nil {
					fmt.Fprintf(app.Err, "Found in %d stores.\n", len(d.Result.Offers))
				}
				return d.Catalog, nil
			}
		case api.DiscoveryEmpty:
			return nil, &exitError{code: ExitNotFound, msg: fmt.Sprintf("no store sells “%s” right now", name)}
		case api.DiscoveryDone:
			return nil, fmt.Errorf("the stores were searched, but “%s” could not be added to the catalog; try again later", name)
		}
		if app.Now().After(deadline) {
			return nil, fmt.Errorf("the store search is still running after %s; run the command again in a moment", o.timeout)
		}
		if err := sleep(ctx, app.PollInterval); err != nil {
			return nil, err
		}
	}
}

// waitForPrices waits until every store of the product was checked once.
func (app *App) waitForPrices(ctx context.Context, client *api.Client, productID string, timeout time.Duration) (*api.Product, error) {
	deadline := app.Now().Add(timeout)
	for {
		products, err := client.Products(ctx)
		if err != nil {
			return nil, err
		}
		for i := range products {
			if products[i].ID == productID && allChecked(products[i]) {
				return &products[i], nil
			}
		}
		if app.Now().After(deadline) {
			for i := range products {
				if products[i].ID == productID {
					fmt.Fprintln(app.Err, "Some stores are not checked yet.")
					return &products[i], nil
				}
			}
			return nil, fmt.Errorf("the product is not in your tracked products")
		}
		if err := sleep(ctx, app.PollInterval); err != nil {
			return nil, err
		}
	}
}

func allChecked(p api.Product) bool { return !waitingForFirstCheck(p) }

// ParseEuros reads a price in euros written the French or English way —
// "450", "449,90", "449.90", "1 299,99 €" — and returns it in cents.
func ParseEuros(s string) (int64, error) {
	clean := strings.NewReplacer(" ", "", " ", "", " ", "", "€", "", "EUR", "", "eur", "").Replace(strings.TrimSpace(s))
	clean = strings.Replace(clean, ",", ".", 1)
	whole, frac, hasFrac := strings.Cut(clean, ".")
	if whole == "" || !digits(whole) || (hasFrac && (frac == "" || len(frac) > 2 || !digits(frac))) {
		return 0, fmt.Errorf("“%s” is not a price; write it like 450 or 449,90", s)
	}
	for len(frac) < 2 {
		frac += "0"
	}
	cents, err := strconv.ParseInt(whole+frac, 10, 64)
	if err != nil || cents <= 0 {
		return 0, fmt.Errorf("“%s” is not a price above 0", s)
	}
	return cents, nil
}

func digits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
