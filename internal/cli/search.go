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
	var ptype string
	var limit int
	cmd := &cobra.Command{
		Use:   "search [QUERY]",
		Short: "Search Pricewatch's catalog",
		Long: `Lists the catalog products whose name has every word of QUERY (case and
accents ignored), best match first, with the number of stores that sell each
one. ✓ marks the products you track. Without QUERY, lists the catalog.

Track a result with: pricewatch add --id ID`,
		Example: `  pricewatch search "elden ring"
  pricewatch search iphone --type smartphone --limit 50
  pricewatch search --type tcg`,
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
			query := strings.TrimSpace(strings.Join(args, " "))
			found, err := client.SearchCatalog(cmd.Context(), query, ptype, limit)
			if err != nil {
				return err
			}
			if asJSON {
				return app.writeJSON(found)
			}
			if len(found) == 0 {
				if query == "" {
					return &exitError{code: ExitNotFound, msg: "the catalog has no product of this type"}
				}
				return &exitError{code: ExitNotFound, msg: fmt.Sprintf(
					"no catalog product matches “%s”; pricewatch add \"%s\" searches the stores for it", query, query)}
			}
			tracked := app.trackedNames(cmd.Context(), client)
			t := render.NewTable(app.Out, "PRODUCT", "TYPE", "STORES", "TRACKED", "ID")
			for _, c := range found {
				mark := ""
				if tracked(c.Name) {
					mark = "✓"
				}
				t.Row(c.Name, typeName(c.Type), fmt.Sprint(len(c.Offers)), mark, c.ID)
			}
			return t.Flush()
		},
	}
	cmd.Flags().StringVar(&ptype, "type", "", "only this product type: "+strings.Join(productTypes, ", "))
	cmd.Flags().IntVar(&limit, "limit", 20, "how many products to list, 1 to 100")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the products as JSON, as the API returns them")
	return cmd
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
