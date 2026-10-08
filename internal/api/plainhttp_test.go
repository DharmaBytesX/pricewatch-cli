package api

import (
	"context"
	"strings"
	"testing"
)

func TestPlainHTTP(t *testing.T) {
	for base, want := range map[string]bool{
		"https://pricewatch.exe.xyz": false,
		"http://pricewatch.exe.xyz":  true,
		"HTTP://192.168.1.20:3000":   true,
		"http://localhost:3000":      false,
		"http://127.0.0.1:8080":      false,
		"http://[::1]:8080":          false,
	} {
		if got := plainHTTP(base); got != want {
			t.Errorf("plainHTTP(%q) = %v, want %v", base, got, want)
		}
	}
}

// The token is not sent to a plain HTTP address of another computer: the
// request fails before anything leaves.
func TestNoTokenOverPlainHTTP(t *testing.T) {
	c := New("http://pricewatch.exe.xyz", "pwt_secret", "test", nil)
	_, err := c.Me(context.Background())
	if err == nil || !strings.Contains(err.Error(), "plain HTTP") {
		t.Fatalf("err = %v", err)
	}
}
