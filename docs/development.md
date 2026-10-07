# Development

## Requirements

- Go 1.27 or later.
- golangci-lint 2.14.0, for `make lint`:
  `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0`.
  `make lint-docker` runs the same version in Docker instead.

## Tasks

| Command | What it does |
| --- | --- |
| `make build` | builds `./pricewatch`, with the version from `git describe` |
| `make test` | runs the tests with the race detector |
| `make lint` | runs golangci-lint with `.golangci.yml` |
| `make docs` | writes the command reference in `docs/commands` from the commands' help |
| `make check` | lint, tests, and a check that `docs/commands` is up to date: what CI runs |

## Project layout

| Path | What it holds |
| --- | --- |
| `cmd/pricewatch` | `main`: runs `cli.Execute` and exits with its code |
| `internal/cli` | the commands, one file each (`auth.go`, `prices.go`, `add.go`, `version.go`); `app.go` holds the command tree, the exit codes and the error messages |
| `internal/api` | the client of Pricewatch's `/api/v1`, its types and its errors |
| `internal/config` | the configuration file, and the order of flags, environment and file |
| `internal/render` | prices, times, conditions and tables, written as on the website |
| `internal/match` | product-name matching: every word, case and accents ignored |
| `internal/fakeapi` | an in-memory `/api/v1` for the tests |
| `internal/version` | the version, set at build time |
| `tools/gendocs` | writes `docs/commands` |

## How a command works

A command is a function `newXCommand(app *App, g *globals) *cobra.Command`
in its own file, added in `NewRootCommand` (`internal/cli/app.go`).

- It reads and writes through `App` (`In`, `Out`, `Err`, `Getenv`, `Now`,
  `HTTPClient`), never through `os` directly, so tests run it.
- It gets its API client from `app.client(g)`, which applies the address and
  token settings and refuses to run without a token.
- Results go to `app.Out`; progress and questions go to `app.Err`, so
  `--json` output stays parseable.
- It returns errors; `describe` (`app.go`) turns them into the message and
  the exit code. `usageError` is exit code 2; `exitError` gives any code.
  API errors are mapped by HTTP status: 401 and 403 are code 4, 404 is
  code 3.

To add a command:

1. Write `internal/cli/<name>.go` with `newNameCommand`, and add it in
   `NewRootCommand`.
2. Add the API calls it needs in `internal/api/client.go`, and the routes to
   `internal/fakeapi`.
3. Test it in `internal/cli/cli_test.go` with the `harness`: it runs the
   command against the fake server and returns standard output, standard
   error and the exit code.
4. Run `make docs` and commit `docs/commands`.

## Tests and the real API

`internal/fakeapi` follows the rules of Pricewatch's `/api/v1` that the CLI
depends on: tokens and scopes, catalog search, discoveries, tracking, the
plan limit. When `/api/v1` changes in Pricewatch, update the fake in the same
way. Pricewatch's own tests cover the real API
(`backend/internal/api/v1_integration_test.go` in
[DharmaBytesX/pricewatch](https://github.com/DharmaBytesX/pricewatch)).

The API only adds fields and routes; it never renames or removes them, so
released versions of the CLI keep working. The types in `internal/api`
ignore fields they do not know.

## Continuous integration

`.github/workflows/ci.yml` runs on every push to `main` and every pull
request:

- tests on Linux (with the race detector) and Windows, after `go mod verify`
  and `go vet`;
- golangci-lint 2.14.0;
- `make check-docs`;
- a build for every release target (Linux, macOS, Windows; x86-64, ARM64).

## Releases

Push a version tag: `git tag v1.0.0 && git push origin v1.0.0`.
`.github/workflows/release.yml` runs the tests, then GoReleaser
(`.goreleaser.yaml`) builds the archives, writes `checksums.txt`, and
publishes the GitHub release with notes generated from the commits.
