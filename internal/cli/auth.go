package cli

import (
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/DharmaBytesX/pricewatch-cli/internal/api"
	"github.com/DharmaBytesX/pricewatch-cli/internal/config"
	"github.com/DharmaBytesX/pricewatch-cli/internal/render"
)

// tokenPrefix starts every Pricewatch API token.
const tokenPrefix = "pwt_"

func newAuthCommand(app *App, g *globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Sign in with an API token, and check which one is used",
		Long: `Create an API token on Pricewatch's Settings page ("API tokens"): choose its
access ("Read", or "Read and add"; "Manage the webhook" for the webhook
commands) and when it expires. "pricewatch auth login" saves it in the
configuration file.

The token is taken from, in order: the --token flag, the PRICEWATCH_TOKEN
variable, then the configuration file.`,
	}
	cmd.AddCommand(newLoginCommand(app, g), newStatusCommand(app, g), newLogoutCommand(app, g))
	return cmd
}

func newLoginCommand(app *App, g *globals) *cobra.Command {
	var withToken bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Save an API token, after checking it with Pricewatch",
		Example: `  pricewatch auth login                          # paste the token when asked
  pricewatch auth login --with-token < token.txt # read it from standard input`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			token, err := app.readToken(withToken)
			if err != nil {
				return err
			}
			if !strings.HasPrefix(token, tokenPrefix) {
				return usageError("this is not a Pricewatch API token: they start with " + tokenPrefix)
			}
			s, path, f, err := app.settings(g)
			if err != nil {
				return err
			}
			me, err := app.newClient(s.URL, token).Me(cmd.Context())
			if api.StatusOf(err) == 401 {
				return &exitError{code: ExitDenied, msg: "Pricewatch refused this token: it is unknown, expired or revoked"}
			}
			if err != nil {
				return err
			}
			f.Token = token
			if g.url != "" {
				f.URL = s.URL // remember the address given with this login
			}
			if err := config.Save(path, f); err != nil {
				return err
			}
			fmt.Fprintf(app.Out, "Signed in to %s as %s (%s plan).\n", s.URL, me.Email, planName(me.Plan))
			if me.Token != nil {
				fmt.Fprintf(app.Out, "Token “%s”: %s, expires on %s.\n", me.Token.Name, scopeText(me.Token.Scopes), render.Date(me.Token.ExpiresAt))
			}
			fmt.Fprintf(app.Out, "Saved in %s.\n", path)
			if app.Getenv(config.EnvToken) != "" {
				fmt.Fprintf(app.Err, "Note: %s is set, and is used instead of the saved token.\n", config.EnvToken)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&withToken, "with-token", false, "read the token from standard input")
	return cmd
}

// readToken reads a token from standard input, or asks for it.
func (app *App) readToken(fromStdin bool) (string, error) {
	switch {
	case fromStdin:
		b, err := io.ReadAll(io.LimitReader(app.In, 4096))
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(strings.SplitN(string(b), "\n", 2)[0]), nil
	case app.StdinIsTerminal():
		fmt.Fprint(app.Err, "Paste an API token (create one on Pricewatch's Settings page): ")
		token, err := app.ReadSecret()
		fmt.Fprintln(app.Err)
		return strings.TrimSpace(token), err
	}
	return "", usageError("no terminal to ask for the token: give it on standard input with --with-token, " +
		"e.g. pricewatch auth login --with-token < token.txt")
}

func newStatusCommand(app *App, g *globals) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the account, the token and the address in use",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, s, err := app.client(g)
			if err != nil {
				return err
			}
			me, err := client.Me(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON {
				return app.writeJSON(me)
			}
			t := render.NewTable(app.Out, "Address", s.URL+" ("+s.URLSource+")")
			t.Row("Account", fmt.Sprintf("%s (%s plan: %s)", me.Email, planName(me.Plan), planText(me.Limits)))
			if me.Token != nil {
				t.Row("Token", fmt.Sprintf("“%s” (%s): %s", me.Token.Name, s.TokenSource, scopeText(me.Token.Scopes)))
				t.Row("Expires", fmt.Sprintf("%s (%s)", render.Date(me.Token.ExpiresAt), daysLeft(me.Token.ExpiresAt, app.Now())))
			}
			if err := t.Flush(); err != nil {
				return err
			}
			if me.Token != nil && me.Token.ExpiresAt.Sub(app.Now()) < 7*24*time.Hour {
				fmt.Fprintln(app.Err, "The token expires soon: create a new one on Pricewatch's Settings page.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the answer as JSON")
	return cmd
}

func newLogoutCommand(app *App, g *globals) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the saved token from the configuration file",
		Long: `Removes the saved token from the configuration file. The token keeps working
until it expires: revoke it on Pricewatch's Settings page to stop it at once.`,
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			_, path, f, err := app.settings(g)
			if err != nil {
				return err
			}
			if f.Token == "" {
				fmt.Fprintln(app.Out, "No token is saved.")
				return nil
			}
			f.Token = ""
			if err := config.Save(path, f); err != nil {
				return err
			}
			fmt.Fprintf(app.Out, "Removed the token from %s. It keeps working until it expires or you revoke it on Pricewatch's Settings page.\n", path)
			return nil
		},
	}
}

func planName(p string) string {
	if p == "" {
		return "unknown"
	}
	return strings.ToUpper(p[:1]) + p[1:]
}

// planText describes a plan's limits: "1 product, checked every 5 minutes".
func planText(p api.Plan) string {
	products := "any number of products"
	if p.MaxProducts == 1 {
		products = "1 product"
	} else if p.MaxProducts > 1 {
		products = fmt.Sprintf("%d products", p.MaxProducts)
	}
	every := time.Duration(p.CheckIntervalSeconds) * time.Second
	switch {
	case every <= 0:
		return products
	case every < time.Minute:
		return fmt.Sprintf("%s, checked every %d seconds", products, int(every.Seconds()))
	case every == time.Minute:
		return products + ", checked every minute"
	}
	return fmt.Sprintf("%s, checked every %d minutes", products, int(every.Minutes()))
}

// Scopes an API token can hold.
const (
	scopeProductsRead  = "products:read"
	scopeProductsWrite = "products:write"
	scopeWebhooks      = "webhooks:write"
)

// scopeText says what a token may do.
func scopeText(scopes []string) string {
	var text string
	switch {
	case contains(scopes, scopeProductsWrite):
		text = "reads and adds products"
	case contains(scopes, scopeProductsRead):
		text = "reads products and prices"
	}
	switch {
	case contains(scopes, scopeWebhooks) && text == "":
		return "manages the webhook; no access to products"
	case contains(scopes, scopeWebhooks):
		return text + " and manages the webhook"
	case text == "":
		return "no access to products"
	}
	return text
}

func daysLeft(t, now time.Time) string {
	d := t.Sub(now)
	if d <= 0 {
		return "expired"
	}
	days := int(math.Ceil(d.Hours() / 24))
	if days == 1 {
		return "in 1 day"
	}
	return fmt.Sprintf("in %d days", days)
}
