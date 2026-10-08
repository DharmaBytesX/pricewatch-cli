package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/DharmaBytesX/pricewatch-cli/internal/api"
	"github.com/DharmaBytesX/pricewatch-cli/internal/match"
	"github.com/DharmaBytesX/pricewatch-cli/internal/render"
)

// typeNames are the product types as the website names them.
var typeNames = map[string]string{
	"video_game": "Game",
	"console":    "Console",
	"smartphone": "Phone",
	"laptop":     "Laptop",
	"tcg":        "Cards",
}

func typeName(t string) string {
	if n := typeNames[t]; n != "" {
		return n
	}
	return "Other"
}

func newSearchCommand(app *App, g *globals) *cobra.Command {
	var asJSON bool
	var ptype, storeName string
	var limit int
	cmd := &cobra.Command{
		Use:   "search [QUERY]",
		Short: "Search Pricewatch's catalog",
		Long: `Lists the catalog products whose name has every word of QUERY (case and
accents ignored), best match first, with the number of stores that sell each
one. ✓ marks the products you track. Without QUERY, lists the catalog.

--store keeps the products one store sells, and shows that store's link:
give its name ("E.Leclerc", "Amazon FR") or its code name ("leclerc",
"amazon").

Track a result with: pricewatch add --id ID`,
		Example: `  pricewatch search "elden ring"
  pricewatch search iphone --type smartphone --limit 50
  pricewatch search --type tcg
  pricewatch search --store leclerc --limit 100`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if ptype != "" && !contains(productTypes, ptype) {
				return usageError("--type must be one of: " + strings.Join(productTypes, ", "))
			}
			if limit < 1 || limit > 100 {
				return usageError("--limit must be between 1 and 100")
			}
			client, _, err := app.client(g)
			if err != nil {
				return err
			}
			var st *api.Store
			if storeName != "" {
				if st, err = app.resolveStore(cmd.Context(), client, storeName); err != nil {
					return err
				}
			}
			query := strings.TrimSpace(strings.Join(args, " "))
			cq := api.CatalogQuery{Text: query, Type: ptype, Limit: limit}
			if st != nil {
				cq.Store = st.Name
			}
			found, err := client.SearchCatalog(cmd.Context(), cq)
			if err != nil {
				return err
			}
			if asJSON {
				return app.writeJSON(found)
			}
			if len(found) == 0 {
				switch {
				case query == "" && st != nil:
					return &exitError{code: ExitNotFound, msg: fmt.Sprintf("no catalog product of this kind is sold by %s", st.DisplayName)}
				case query == "":
					return &exitError{code: ExitNotFound, msg: "the catalog has no product of this type"}
				case st != nil:
					return &exitError{code: ExitNotFound, msg: fmt.Sprintf("%s sells no catalog product matching “%s”", st.DisplayName, query)}
				}
				return &exitError{code: ExitNotFound, msg: fmt.Sprintf(
					"no catalog product matches “%s”; pricewatch add \"%s\" searches the stores for it", query, query)}
			}
			tracked := app.trackedNames(cmd.Context(), client)
			header := []string{"PRODUCT", "TYPE", "STORES", "TRACKED", "ID"}
			if st != nil {
				header = append(header, strings.ToUpper(st.DisplayName)+" LINK")
			}
			t := render.NewTable(app.Out, header...)
			for _, c := range found {
				mark := ""
				if tracked(c.Name) {
					mark = "✓"
				}
				row := []string{c.Name, typeName(c.Type), fmt.Sprint(len(c.Offers)), mark, c.ID}
				if st != nil {
					row = append(row, offerURL(c, st.Name))
				}
				t.Row(row...)
			}
			return t.Flush()
		},
	}
	cmd.Flags().StringVar(&ptype, "type", "", "only this product type: "+strings.Join(productTypes, ", "))
	cmd.Flags().StringVar(&storeName, "store", "", `only the products this store sells, e.g. "E.Leclerc" or leclerc`)
	cmd.Flags().IntVar(&limit, "limit", 20, "how many products to list, 1 to 100")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the products as JSON, as the API returns them")
	return cmd
}

// resolveStore finds a store by its code name ("leclerc") or its name
// ("E.Leclerc"), ignoring case, spaces and punctuation.
func (app *App) resolveStore(ctx context.Context, client *api.Client, name string) (*api.Store, error) {
	list, err := client.Stores(ctx)
	if err != nil {
		return nil, err
	}
	key := func(s string) string {
		return strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				return r
			}
			return -1
		}, match.Fold(s))
	}
	names := make([]string, 0, len(list))
	for i, s := range list {
		if key(s.Name) == key(name) || key(s.DisplayName) == key(name) {
			return &list[i], nil
		}
		names = append(names, s.DisplayName+" ("+s.Name+")")
	}
	return nil, usageError(fmt.Sprintf("unknown store “%s”; the stores are: %s", name, strings.Join(names, ", ")))
}

// offerURL is the product's page at a store.
func offerURL(c api.CatalogProduct, storeName string) string {
	for _, o := range c.Offers {
		if o.Marketplace == storeName {
			return o.URL
		}
	}
	return ""
}

// trackedNames reports whether the user tracks a product of that name.
// When the list cannot be read, nothing is marked.
func (app *App) trackedNames(ctx context.Context, client *api.Client) func(string) bool {
	names := map[string]bool{}
	if products, err := client.Products(ctx); err == nil {
		for _, p := range products {
			names[strings.Join(match.Words(p.Name), " ")] = true
		}
	}
	return func(name string) bool { return names[strings.Join(match.Words(name), " ")] }
}
