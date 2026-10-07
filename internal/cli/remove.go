package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/DharmaBytesX/pricewatch-cli/internal/api"
	"github.com/DharmaBytesX/pricewatch-cli/internal/match"
)

func newRemoveCommand(app *App, g *globals) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "remove PRODUCT",
		Aliases: []string{"rm", "untrack"},
		Short:   "Stop tracking a product and delete its price history",
		Long: `Stops tracking one of your products and deletes its price history. PRODUCT is
any words of the product's name, in any order (case and accents ignored), or
its ID or the first 8 characters of it or more.

When several of your products match, pricewatch lists them and stops: give
the exact name or the ID. It asks before removing; without a terminal, add
--yes.`,
		Example: `  pricewatch remove "elden ring ps5"
  pricewatch remove 0b6f2c1e --yes`,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return usageError("give the product to remove: words of its name, or its ID")
			}
			return nil
		},
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
			p, err := pickProduct(findProducts(products, query), query)
			if err != nil {
				return err
			}
			if !yes {
				ok, err := app.confirm(fmt.Sprintf("Stop tracking “%s” (%d stores) and delete its price history? [y/N] ", p.Name, len(p.Watches)))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(app.Out, "Nothing removed.")
					return nil
				}
			}
			if err := client.DeleteProduct(cmd.Context(), p.ID); err != nil {
				return err
			}
			fmt.Fprintf(app.Out, "Stopped tracking %s.\n", p.Name)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "remove without asking")
	return cmd
}

// pickProduct returns the one product to act on: the only match, or the
// one named exactly like the query.
func pickProduct(found []api.Product, query string) (*api.Product, error) {
	switch len(found) {
	case 0:
		return nil, &exitError{code: ExitNotFound, msg: fmt.Sprintf("you track no product matching “%s”", query)}
	case 1:
		return &found[0], nil
	}
	for i := range found {
		if match.Same(found[i].Name, query) {
			return &found[i], nil
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "“%s” matches %d of your products:\n", query, len(found))
	for _, p := range found {
		fmt.Fprintf(&b, "  %s  (%s)\n", p.Name, p.ID)
	}
	b.WriteString("Give the exact name or the ID")
	return nil, usageError(b.String())
}

// confirm asks a yes/no question on the terminal; without one, it refuses
// and asks for --yes.
func (app *App) confirm(question string) (bool, error) {
	if !app.StdinIsTerminal() {
		return false, usageError("no terminal to confirm: add --yes")
	}
	fmt.Fprint(app.Err, question)
	answer, err := app.readLine()
	if err != nil {
		return false, nil
	}
	switch strings.ToLower(answer) {
	case "y", "yes", "o", "oui":
		return true, nil
	}
	return false, nil
}
