package cli

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cobra"

	"github.com/DharmaBytesX/pricewatch-cli/internal/api"
	"github.com/DharmaBytesX/pricewatch-cli/internal/render"
)

// recentDeliveries is how many deliveries "pricewatch webhook" lists.
const recentDeliveries = 10

func newWebhookCommand(app *App, g *globals) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Send your alerts to your own service, and check what was sent",
		Long: `Pricewatch can send each of your alerts (a price drop, a product back in
stock) to an HTTPS address of yours, as a signed JSON request, in addition
to Discord. Your service, or an automation tool that receives webhooks,
then acts on it.

Without a command, shows the webhook's address and its 10 latest
deliveries: the event, whether it was sent, the answer of your service
(its HTTP status, or why there was none), the number of attempts, and when.

Showing the webhook needs an API token with "Read" access; setting,
testing or removing it needs "Write" access. The event body and how to
check the signature are described on Pricewatch's docs page, at
/docs#webhooks.`,
		Example: `  pricewatch webhook set https://example.com/pricewatch
  pricewatch webhook test
  pricewatch webhook
  pricewatch webhook --json | jq '.deliveries[] | select(.status == "failed")'`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := app.client(g)
			if err != nil {
				return err
			}
			hook, err := client.Webhook(cmd.Context())
			if err != nil {
				return webhookError(err)
			}
			if asJSON {
				deliveries, err := client.WebhookDeliveries(cmd.Context(), recentDeliveries)
				if err != nil {
					return webhookError(err)
				}
				if deliveries == nil {
					deliveries = []api.WebhookDelivery{}
				}
				return app.writeJSON(struct {
					URL        *string               `json:"url"`
					Deliveries []api.WebhookDelivery `json:"deliveries"`
				}{hook.URL, deliveries})
			}
			if hook.URL == nil {
				fmt.Fprintln(app.Out, "No webhook is set. Set one with: pricewatch webhook set https://…")
				return nil
			}
			deliveries, err := client.WebhookDeliveries(cmd.Context(), recentDeliveries)
			if err != nil {
				return webhookError(err)
			}
			fmt.Fprintf(app.Out, "Alerts are sent to %s\n\n", *hook.URL)
			if len(deliveries) == 0 {
				fmt.Fprintln(app.Out, "Nothing sent yet. Send a test event with: pricewatch webhook test")
				return nil
			}
			t := render.NewTable(app.Out, "TYPE", "STATUS", "ANSWER", "ATTEMPTS", "WHEN")
			for _, d := range deliveries {
				t.Row(eventName(d.Type), d.Status, answerText(d), fmt.Sprint(d.Attempts), render.Ago(&d.CreatedAt, app.Now()))
			}
			return t.Flush()
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, `print the address and the deliveries as JSON: {"url": …, "deliveries": […]}`)
	cmd.AddCommand(
		newWebhookSetCommand(app, g),
		newWebhookTestCommand(app, g),
		newWebhookSecretCommand(app, g),
		newWebhookRemoveCommand(app, g),
	)
	return cmd
}

func newWebhookSetCommand(app *App, g *globals) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "set URL",
		Short: "Send your alerts to this address",
		Long: `Sends your alerts to URL, which must start with https:// and be reachable from
the internet. Pricewatch sends a POST request with a JSON body for each
alert.

For a new webhook, Pricewatch creates a signing secret and pricewatch prints
it once: Pricewatch signs every request with it (the webhook-signature
header), so your service can check that the request comes from Pricewatch.
Copy it then; Pricewatch does not show it again, and "pricewatch webhook
new-secret" replaces it. Changing the address keeps the secret.

In a script, --json prints the answer, and jq -r .secret gives the secret
alone.`,
		Example: `  pricewatch webhook set https://example.com/pricewatch
  pricewatch webhook set https://hooks.example.com/abc123 --json | jq -r .secret > secret.txt`,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageError("give the address to send the alerts to, e.g. pricewatch webhook set https://example.com/pricewatch")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			u, err := url.Parse(args[0])
			if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
				return usageError(fmt.Sprintf("“%s” is not a web address: give the whole address, starting with https://", args[0]))
			}
			client, _, err := app.client(g)
			if err != nil {
				return err
			}
			hook, err := client.SetWebhook(cmd.Context(), args[0])
			if err != nil {
				return webhookError(err)
			}
			if asJSON {
				return app.writeJSON(hook)
			}
			address := args[0]
			if hook.URL != nil {
				address = *hook.URL
			}
			if hook.Secret == "" {
				fmt.Fprintf(app.Out, "Alerts are now sent to %s. The signing secret did not change.\n", address)
				return nil
			}
			fmt.Fprintf(app.Out, "Alerts are now sent to %s.\n\n", address)
			printSecret(app, hook.Secret)
			fmt.Fprintln(app.Out, "Send a test event with: pricewatch webhook test")
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, `print the answer as JSON: {"url": …, "secret": …}; the secret only for a new webhook`)
	return cmd
}

func newWebhookTestCommand(app *App, g *globals) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Send a test event to the webhook now",
		Long: `Sends a test event to the webhook now, and shows your service's answer. A test
event has the same fields as an alert, with example values, and the type
"test". The exit code is 1 when the event was not delivered.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := app.client(g)
			if err != nil {
				return err
			}
			d, err := client.TestWebhook(cmd.Context())
			if err != nil {
				return webhookError(err)
			}
			if asJSON {
				if err := app.writeJSON(struct {
					Delivery *api.WebhookDelivery `json:"delivery"`
				}{d}); err != nil {
					return err
				}
			}
			if d.Status != api.DeliverySent {
				return &exitError{code: ExitFailure, msg: "the test event was not delivered: " + failureText(*d)}
			}
			if !asJSON {
				fmt.Fprintf(app.Out, "Sent: %s%s (event %s).\n", answerText(*d), took(*d), d.EventID)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, `print the delivery as JSON: {"delivery": {…}}`)
	return cmd
}

func newWebhookSecretCommand(app *App, g *globals) *cobra.Command {
	var asJSON, yes bool
	cmd := &cobra.Command{
		Use:   "new-secret",
		Short: "Replace the webhook's signing secret",
		Long: `Replaces the secret that signs the webhook's requests, and prints the new one
once. Pricewatch signs the next requests with the new secret: give it to
your service. It asks before replacing; without a terminal, add --yes.`,
		Example: `  pricewatch webhook new-secret
  pricewatch webhook new-secret --yes --json | jq -r .secret > secret.txt`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := app.client(g)
			if err != nil {
				return err
			}
			if !yes {
				ok, err := app.confirm("Replace the signing secret? The next requests are signed with the new one. [y/N] ")
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(app.Out, "The secret did not change.")
					return nil
				}
			}
			secret, err := client.NewWebhookSecret(cmd.Context())
			if err != nil {
				return webhookError(err)
			}
			if asJSON {
				return app.writeJSON(map[string]string{"secret": secret})
			}
			printSecret(app, secret)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, `print the answer as JSON: {"secret": …}`)
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "replace without asking")
	return cmd
}

func newWebhookRemoveCommand(app *App, g *globals) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Stop sending alerts to the webhook",
		Long: `Removes the webhook and its signing secret: Pricewatch stops sending alerts to
it. It asks before removing; without a terminal, add --yes.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := app.client(g)
			if err != nil {
				return err
			}
			hook, err := client.Webhook(cmd.Context())
			if err != nil {
				return webhookError(err)
			}
			if hook.URL == nil {
				fmt.Fprintln(app.Out, "No webhook is set.")
				return nil
			}
			if !yes {
				ok, err := app.confirm(fmt.Sprintf("Stop sending alerts to %s and delete its secret? [y/N] ", *hook.URL))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(app.Out, "Nothing removed.")
					return nil
				}
			}
			if err := client.RemoveWebhook(cmd.Context()); err != nil {
				return webhookError(err)
			}
			fmt.Fprintf(app.Out, "Alerts are not sent to %s any more.\n", *hook.URL)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "remove without asking")
	return cmd
}

func printSecret(app *App, secret string) {
	fmt.Fprintf(app.Out, "Signing secret: %s\n", secret)
	fmt.Fprintln(app.Out, `Pricewatch signs every request with it (the webhook-signature header), so that
your service can check that a request comes from Pricewatch. Copy it now:
Pricewatch does not show it again. "pricewatch webhook new-secret" replaces it.`)
}

// webhookError explains the errors of the webhook routes; others go to
// describe.
func webhookError(err error) error {
	status := api.StatusOf(err)
	switch {
	case status == http.StatusNotFound && api.FromPricewatch(err):
		return &exitError{code: ExitNotFound, msg: err.Error() + ": set one with: pricewatch webhook set https://…"}
	case status == http.StatusNotFound:
		return &exitError{code: ExitFailure, msg: err.Error() + ": this Pricewatch server may not have webhooks yet"}
	case status == http.StatusBadRequest:
		return usageError(err.Error())
	}
	return err
}

var eventNames = map[string]string{
	"price_drop": "price drop",
	"restock":    "back in stock",
	"test":       "test",
}

func eventName(t string) string {
	if n := eventNames[t]; n != "" {
		return n
	}
	return t
}

// answerText is the answer of the user's service: its HTTP status, or why
// there was none.
func answerText(d api.WebhookDelivery) string {
	switch {
	case d.ResponseStatus != nil:
		return fmt.Sprintf("HTTP %d", *d.ResponseStatus)
	case d.Error != nil && *d.Error != "":
		return shorten(*d.Error, 60)
	}
	return "—"
}

// failureText says why a delivery failed: the answer, and the reason when
// it says more.
func failureText(d api.WebhookDelivery) string {
	answer := answerText(d)
	if d.ResponseStatus != nil && d.Error != nil && *d.Error != "" {
		return answer + " (" + *d.Error + ")"
	}
	if d.ResponseStatus == nil && d.Error != nil && *d.Error != "" {
		return *d.Error
	}
	return answer
}

// took is how long the delivery took, " in 120 ms", when the server says
// when it was sent.
func took(d api.WebhookDelivery) string {
	if d.DeliveredAt == nil || d.DeliveredAt.Before(d.CreatedAt) {
		return ""
	}
	elapsed := d.DeliveredAt.Sub(d.CreatedAt)
	if elapsed < time.Second {
		return fmt.Sprintf(" in %d ms", elapsed.Milliseconds())
	}
	return fmt.Sprintf(" in %.1f s", elapsed.Seconds())
}

func shorten(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
