// Package cli implements the pricewatch commands.
//
// Each command lives in its own file and talks to Pricewatch through
// internal/api. Commands use only what App gives them (streams, environment,
// clock, HTTP client), so tests run them against a fake server.
package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/DharmaBytesX/pricewatch-cli/internal/api"
	"github.com/DharmaBytesX/pricewatch-cli/internal/config"
	"github.com/DharmaBytesX/pricewatch-cli/internal/version"
)

// Exit codes. Scripts can rely on them.
const (
	ExitOK       = 0
	ExitFailure  = 1 // the request failed: network, server, or an unexpected answer
	ExitUsage    = 2 // the command line is wrong or ambiguous
	ExitNotFound = 3 // no tracked product matches, or no store sells the product
	ExitDenied   = 4 // not signed in; token unknown, expired, revoked or missing a scope; plan limit
)

// App is what the commands use from the outside world.
type App struct {
	In     io.Reader
	Out    io.Writer
	Err    io.Writer
	Getenv func(string) string
	// HTTPClient sends the API requests; nil uses a client with
	// api.DefaultTimeout.
	HTTPClient *http.Client
	Now        func() time.Time
	// StdinIsTerminal reports whether In is a terminal: commands ask
	// questions only then.
	StdinIsTerminal func() bool
	// ReadSecret reads a line from the terminal without showing it.
	ReadSecret func() (string, error)
	// PollInterval is how often commands check progress (discoveries, first
	// price checks).
	PollInterval time.Duration

	stdin *bufio.Reader
}

// NewApp returns an App on the process's streams and environment.
func NewApp() *App {
	return &App{
		In:              os.Stdin,
		Out:             os.Stdout,
		Err:             os.Stderr,
		Getenv:          os.Getenv,
		Now:             time.Now,
		StdinIsTerminal: func() bool { return term.IsTerminal(int(os.Stdin.Fd())) },
		ReadSecret: func() (string, error) {
			b, err := term.ReadPassword(int(os.Stdin.Fd()))
			return string(b), err
		},
		PollInterval: 500 * time.Millisecond,
	}
}

// Execute runs the CLI with args (without the program name) and returns the
// process's exit code.
func Execute(app *App, args []string) int {
	root := NewRootCommand(app)
	root.SetArgs(args)
	err := root.Execute()
	if err == nil {
		return ExitOK
	}
	code, msg := describe(err)
	fmt.Fprintln(app.Err, "pricewatch: "+msg)
	return code
}

// globals are the flags every command accepts.
type globals struct {
	url   string
	token string
}

// NewRootCommand builds the command tree.
func NewRootCommand(app *App) *cobra.Command {
	g := &globals{}
	root := &cobra.Command{
		Use:   "pricewatch",
		Short: "Follow product prices and stock in French stores",
		Long: `pricewatch shows the prices of the products you track on Pricewatch, and adds
products to track. It uses an API token: create one on Pricewatch's Settings
page, then run "pricewatch auth login".`,
		Version:       version.String(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetIn(app.In)
	root.SetOut(app.Out)
	root.SetErr(app.Err)
	root.SetVersionTemplate("pricewatch {{.Version}}\n")
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageError(err.Error()) })
	root.PersistentFlags().StringVar(&g.url, "url", "",
		"Pricewatch address (default: $"+config.EnvURL+", the config file, or "+config.DefaultURL+")")
	root.PersistentFlags().StringVar(&g.token, "token", "",
		"API token (default: $"+config.EnvToken+" or the config file); other users of the computer can see a flag, prefer the variable")

	root.AddCommand(
		newAuthCommand(app, g),
		newPricesCommand(app, g),
		newAddCommand(app, g),
		newVersionCommand(app),
	)
	return root
}

// settings resolves the address and token, and returns the config file.
func (app *App) settings(g *globals) (config.Settings, string, config.File, error) {
	path, err := config.Path(app.Getenv)
	if err != nil {
		return config.Settings{}, "", config.File{}, err
	}
	f, err := config.Load(path)
	if err != nil {
		return config.Settings{}, path, f, err
	}
	return config.Resolve(g.url, g.token, app.Getenv, f), path, f, nil
}

// client returns an API client for a command that needs a token.
func (app *App) client(g *globals) (*api.Client, config.Settings, error) {
	s, _, _, err := app.settings(g)
	if err != nil {
		return nil, s, err
	}
	if s.Token == "" {
		return nil, s, &exitError{code: ExitDenied,
			msg: "not signed in: run \"pricewatch auth login\", or set " + config.EnvToken}
	}
	return app.newClient(s.URL, s.Token), s, nil
}

func (app *App) newClient(url, token string) *api.Client {
	return api.New(url, token, version.UserAgent(), app.HTTPClient)
}

// writeJSON prints v as indented JSON on standard output.
func (app *App) writeJSON(v any) error {
	enc := json.NewEncoder(app.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// readLine reads one line from standard input.
func (app *App) readLine() (string, error) {
	if app.stdin == nil {
		app.stdin = bufio.NewReader(app.In)
	}
	line, err := app.stdin.ReadString('\n')
	if err != nil && (!errors.Is(err, io.EOF) || line == "") {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// sleep waits for d, or until ctx ends.
func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// exitError is an error with its exit code and the message to print.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

func usageError(msg string) error { return &exitError{code: ExitUsage, msg: msg} }

// describe turns an error into an exit code and a message for the user.
func describe(err error) (int, string) {
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.code, ee.msg
	}
	switch status, code := api.StatusOf(err), api.CodeOf(err); {
	case status == http.StatusUnauthorized:
		return ExitDenied, err.Error() + `: create a token on Pricewatch's Settings page, then run "pricewatch auth login"`
	case status == http.StatusForbidden && code == api.CodeInsufficientScope:
		return ExitDenied, err.Error() + `: create a token with "Read and add" access on Pricewatch's Settings page`
	case status == http.StatusForbidden:
		return ExitDenied, err.Error()
	case status == http.StatusNotFound:
		return ExitNotFound, err.Error()
	}
	if strings.HasPrefix(err.Error(), "unknown command") || strings.Contains(err.Error(), "arg(s)") {
		return ExitUsage, err.Error()
	}
	return ExitFailure, err.Error()
}
