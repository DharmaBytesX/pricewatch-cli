package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DharmaBytesX/pricewatch-cli/internal/api"
	"github.com/DharmaBytesX/pricewatch-cli/internal/config"
	"github.com/DharmaBytesX/pricewatch-cli/internal/fakeapi"
)

func TestWebhookSetTestShowRemove(t *testing.T) {
	h := newHarness(t).signedIn(fakeapi.WebhookToken)

	out, errOut, code := h.run("webhook")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "No webhook is set. Set one with: pricewatch webhook set https://")

	// A new webhook: its secret is printed once.
	out, errOut, code = h.run("webhook", "set", "https://example.com/hook")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "Alerts are now sent to https://example.com/hook.", "Signing secret: "+h.srv.WebhookSecret,
		"webhook-signature", "does not show it again", "pricewatch webhook new-secret", "pricewatch webhook test")
	if !strings.HasPrefix(h.srv.WebhookSecret, "whsec_") {
		t.Fatalf("secret %q", h.srv.WebhookSecret)
	}
	if last := h.srv.Log()[len(h.srv.Log())-1]; last != `PUT /api/v1/webhook {"url":"https://example.com/hook"}` {
		t.Fatalf("request %s", last)
	}

	// A new address keeps the secret, which is not printed again.
	secret := h.srv.WebhookSecret
	out, errOut, code = h.run("webhook", "set", "https://example.com/other")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "Alerts are now sent to https://example.com/other. The signing secret did not change.")
	if strings.Contains(out, "whsec_") || h.srv.WebhookSecret != secret {
		t.Fatalf("secret changed or printed: %s", out)
	}

	out, errOut, code = h.run("webhook")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "Alerts are sent to https://example.com/other", "Nothing sent yet. Send a test event with: pricewatch webhook test")

	out, errOut, code = h.run("webhook", "test")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "Sent: HTTP 200 in 120 ms (event msg_test")

	out, errOut, code = h.run("webhook")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "TYPE", "STATUS", "ANSWER", "ATTEMPTS", "WHEN", "test", "sent", "HTTP 200")

	out, errOut, code = h.run("webhook", "--json")
	expect(t, code, ExitOK, out, errOut)
	var got struct {
		URL        *string               `json:"url"`
		Deliveries []api.WebhookDelivery `json:"deliveries"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil || got.URL == nil || *got.URL != "https://example.com/other" ||
		len(got.Deliveries) != 1 || got.Deliveries[0].Type != "test" {
		t.Fatalf("%v: %s", err, out)
	}

	// Removing asks first; without a terminal, --yes is needed.
	_, errOut, code = h.run("webhook", "remove")
	expect(t, code, ExitUsage, "", errOut)
	mustContain(t, errOut, "add --yes")

	h.terminal, h.stdin = true, "n\n"
	out, errOut, code = h.run("webhook", "remove")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, errOut, "Stop sending alerts to https://example.com/other and delete its secret? [y/N]")
	mustContain(t, out, "Nothing removed.")
	if h.srv.WebhookURL == nil {
		t.Fatal("removed after no")
	}

	h.stdin = "y\n"
	out, errOut, code = h.run("webhook", "remove")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "Alerts are not sent to https://example.com/other any more.")
	if h.srv.WebhookURL != nil || h.srv.WebhookSecret != "" {
		t.Fatalf("webhook left: %v %q", h.srv.WebhookURL, h.srv.WebhookSecret)
	}

	h.terminal = false
	out, errOut, code = h.run("webhook", "remove", "--yes")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "No webhook is set.")
}

func TestWebhookDeliveries(t *testing.T) {
	h := newHarness(t).signedIn(fakeapi.WebhookToken)
	address := "https://example.com/hook"
	h.srv.WebhookURL = &address
	status200, status500 := 200, 500
	refused, answered := "dial tcp 203.0.113.9:443: connect: connection refused", "the webhook answered 500"
	alert := int64(4512)
	sent := now.Add(-5*time.Minute + 300*time.Millisecond)
	h.srv.Deliveries = []api.WebhookDelivery{
		{ID: 3, EventID: "msg_3", Type: "price_drop", AlertID: &alert, Status: api.DeliverySent, Attempts: 1,
			ResponseStatus: &status200, CreatedAt: now.Add(-5 * time.Minute), DeliveredAt: &sent},
		{ID: 2, EventID: "msg_2", Type: "restock", Status: api.DeliveryPending, Attempts: 2,
			ResponseStatus: &status500, Error: &answered, CreatedAt: now.Add(-40 * time.Minute)},
		{ID: 1, EventID: "msg_1", Type: "restock", Status: api.DeliveryFailed, Attempts: 6,
			Error: &refused, CreatedAt: now.Add(-3 * time.Hour)},
	}
	out, errOut, code := h.run("webhook")
	expect(t, code, ExitOK, out, errOut)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 6 {
		t.Fatalf("want the address, a blank line, a header and 3 rows:\n%s", out)
	}
	for i, want := range [][]string{
		{"TYPE", "STATUS", "ANSWER", "ATTEMPTS", "WHEN"},
		{"price drop", "sent", "HTTP 200", "1", "5m ago"},
		{"back in stock", "pending", "HTTP 500", "2", "40m ago"},
		{"back in stock", "failed", "dial tcp 203.0.113.9:443: connect: connection refused", "6", "3h ago"},
	} {
		mustContain(t, lines[2+i], want...)
	}
	if !strings.Contains(strings.Join(h.srv.Log(), "\n"), "GET /api/v1/webhook/deliveries?limit=10") {
		t.Fatalf("requests: %v", h.srv.Log())
	}
}

func TestWebhookTestFailures(t *testing.T) {
	h := newHarness(t).signedIn(fakeapi.WebhookToken)
	address := "https://example.com/hook"
	h.srv.WebhookURL = &address

	h.srv.TestStatus = 500
	out, errOut, code := h.run("webhook", "test")
	expect(t, code, ExitFailure, out, errOut)
	mustContain(t, errOut, "the test event was not delivered: HTTP 500 (the webhook answered 500)")

	h.srv.TestStatus, h.srv.TestError = -1, "timeout: no answer within 10 s"
	out, errOut, code = h.run("webhook", "test")
	expect(t, code, ExitFailure, out, errOut)
	mustContain(t, errOut, "the test event was not delivered: timeout: no answer within 10 s")

	// --json prints the delivery, and the exit code still says it failed.
	out, errOut, code = h.run("webhook", "test", "--json")
	expect(t, code, ExitFailure, out, errOut)
	var got struct {
		Delivery api.WebhookDelivery `json:"delivery"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil || got.Delivery.Status != api.DeliveryFailed || got.Delivery.ResponseStatus != nil {
		t.Fatalf("%v: %s", err, out)
	}
}

func TestWebhookNewSecret(t *testing.T) {
	h := newHarness(t).signedIn(fakeapi.WebhookToken)
	if _, errOut, code := h.run("webhook", "set", "https://example.com/hook"); code != ExitOK {
		t.Fatal(errOut)
	}
	first := h.srv.WebhookSecret

	_, errOut, code := h.run("webhook", "new-secret")
	expect(t, code, ExitUsage, "", errOut)
	mustContain(t, errOut, "add --yes")

	h.terminal, h.stdin = true, "n\n"
	out, errOut, code := h.run("webhook", "new-secret")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, errOut, "Replace the signing secret? The next requests are signed with the new one. [y/N]")
	mustContain(t, out, "The secret did not change.")
	if h.srv.WebhookSecret != first {
		t.Fatal("replaced after no")
	}

	h.stdin = "y\n"
	out, errOut, code = h.run("webhook", "new-secret")
	expect(t, code, ExitOK, out, errOut)
	if h.srv.WebhookSecret == first {
		t.Fatal("secret not replaced")
	}
	mustContain(t, out, "Signing secret: "+h.srv.WebhookSecret, "does not show it again")

	h.terminal = false
	out, errOut, code = h.run("webhook", "new-secret", "--yes", "--json")
	expect(t, code, ExitOK, out, errOut)
	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil || got["secret"] != h.srv.WebhookSecret {
		t.Fatalf("%v: %s", err, out)
	}
}

func TestWebhookSetForScripts(t *testing.T) {
	h := newHarness(t).signedIn(fakeapi.WebhookToken)
	out, errOut, code := h.run("webhook", "set", "https://example.com/hook", "--json")
	expect(t, code, ExitOK, out, errOut)
	var got api.Webhook
	if err := json.Unmarshal([]byte(out), &got); err != nil || got.Secret != h.srv.WebhookSecret || *got.URL != "https://example.com/hook" {
		t.Fatalf("%v: %s", err, out)
	}
	// Changing the address: no secret in the answer.
	out, errOut, code = h.run("webhook", "set", "https://example.com/other", "--json")
	expect(t, code, ExitOK, out, errOut)
	if strings.Contains(out, "secret") {
		t.Fatalf("secret in %s", out)
	}
}

func TestWebhookRefusals(t *testing.T) {
	h := newHarness(t).signedIn(fakeapi.WebhookToken)
	for _, args := range [][]string{
		{"webhook", "set"},
		{"webhook", "set", "a", "b"},
		{"webhook", "set", "example.com/hook"},
		{"webhook", "set", "ftp://example.com/hook"},
		{"webhook", "nonsense"},
		{"webhook", "test", "extra"},
	} {
		if _, errOut, code := h.run(args...); code != ExitUsage {
			t.Errorf("%v: exit code %d, want %d (%s)", args, code, ExitUsage, errOut)
		}
	}
	// The server refuses the address: its reason is shown.
	_, errOut, code := h.run("webhook", "set", "http://example.com/hook")
	expect(t, code, ExitUsage, "", errOut)
	mustContain(t, errOut, "the webhook address must start with https://")

	// Without a webhook, there is nothing to test or to sign.
	for _, args := range [][]string{{"webhook", "test"}, {"webhook", "new-secret", "--yes"}} {
		_, errOut, code := h.run(args...)
		expect(t, code, ExitNotFound, "", errOut)
		mustContain(t, errOut, "no webhook is set: set one with: pricewatch webhook set https://")
	}
}

func TestWebhookNeedsWriteToChange(t *testing.T) {
	// A read token shows the webhook, and cannot change it.
	h := newHarness(t).signedIn(fakeapi.ReadToken)
	out, errOut, code := h.run("webhook")
	expect(t, code, ExitOK, out, errOut)
	for _, args := range [][]string{{"webhook", "set", "https://example.com/hook"}, {"webhook", "test"}} {
		_, errOut, code := h.run(args...)
		expect(t, code, ExitDenied, "", errOut)
		mustContain(t, errOut, "does not have the write scope", `"Write" access`)
	}
	if h.srv.WebhookURL != nil {
		t.Fatal("set without the scope")
	}

	// auth status says what the token may do.
	h.signedIn(fakeapi.WebhookToken)
	out, errOut, code = h.run("auth", "status")
	expect(t, code, ExitOK, out, errOut)
	mustContain(t, out, "“automation” (environment): write access: views and changes everything")
}

func TestWebhookOnAnOlderServer(t *testing.T) {
	// A server without the webhook routes answers 404 in plain text.
	older := httptest.NewServer(http.NotFoundHandler())
	defer older.Close()
	h := newHarness(t).signedIn(fakeapi.WebhookToken)
	h.env[config.EnvURL] = older.URL
	_, errOut, code := h.run("webhook")
	expect(t, code, ExitFailure, "", errOut)
	mustContain(t, errOut, "the server answered 404 Not Found: this Pricewatch server may not have webhooks yet")
}

func TestScopeText(t *testing.T) {
	for _, c := range []struct {
		scopes []string
		want   string
	}{
		{[]string{"write"}, "write access: views and changes everything"},
		{[]string{"read"}, "read access: views everything"},
		{nil, "no access"},
	} {
		if got := scopeText(c.scopes); got != c.want {
			t.Errorf("scopeText(%v) = %q, want %q", c.scopes, got, c.want)
		}
	}
}
