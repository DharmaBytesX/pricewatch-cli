# pricewatch CLI

`pricewatch` shows the prices and stock of the products you track on
[Pricewatch](https://pricewatch.exe.xyz), and adds products to track. It
works with an API token that you create on Pricewatch's Settings page.

```console
$ pricewatch add "iPhone 13 128 Go" --condition any --max-price 450 --wait
Tracking iPhone 13 128 Go in 6 stores (Used & new, alerts at or below 450,00 €).

iPhone 13 128 Go
Used & new · alerts at or below 450,00 € · lowest in stock 239,99 € at Cdiscount

STORE         PRICE     STOCK     CHECKED  LINK
Cdiscount     239,99 €  in stock  1s ago   https://www.cdiscount.com/telephonie/…
Easycash      249,99 €  in stock  1s ago   https://bons-plans.easycash.fr/smartphones/…
Recommerce    299,90 €  in stock  0s ago   https://www.recommerce.com/fr/iphone-13-128go-rouge
Cash Express  319,99 €  in stock  1s ago   https://www.cashexpress.fr/p-391284/…
Carrefour     399,00 €  in stock  2s ago   https://www.carrefour.fr/p/iphone-13-128-go-noir-…
Amazon FR     426,23 €  in stock  0s ago   https://www.amazon.fr/dp/B09V3KN99J
```

Contents:

- [Install](#install)
- [Sign in with an API token](#sign-in-with-an-api-token)
- [Show the prices of your products](#show-the-prices-of-your-products)
- [Search the catalog](#search-the-catalog)
- [Add a product](#add-a-product)
- [Stop tracking a product](#stop-tracking-a-product)
- [Settings: address, token, configuration file](#settings-address-token-configuration-file)
- [Exit codes](#exit-codes)
- [Command reference](docs/commands/pricewatch.md)
- [Development](docs/development.md)

## Install

From a release: download the archive for your system from the repository's
Releases page, and put `pricewatch` (`pricewatch.exe` on Windows) in a folder
of your `PATH`. Releases are built for Linux, macOS and Windows, on x86-64
and ARM64. No release is published yet.

With Go 1.27 or later, from the private repository:

```sh
export GOPRIVATE=github.com/DharmaBytesX/*
go install github.com/DharmaBytesX/pricewatch-cli/cmd/pricewatch@latest
```

From the source: `make build` writes `./pricewatch`.

## Sign in with an API token

1. On Pricewatch, open Settings, then "API tokens". Give the token a name,
   choose its access and when it expires (7, 30, 90 days or 1 year), and
   create it.
2. Copy the token. Pricewatch shows it once.
3. Run `pricewatch auth login` and paste it. pricewatch checks it with
   Pricewatch, then saves it in its configuration file.

```console
$ pricewatch auth login
Paste an API token (create one on Pricewatch's Settings page):
Signed in to https://pricewatch.exe.xyz as you@example.com (Premium plan).
Token “My laptop”: reads and adds products, expires on 6 Nov 2026.
Saved in /home/you/.config/pricewatch/config.json.
```

In a script, give the token on standard input:
`pricewatch auth login --with-token < token.txt`, or set
`PRICEWATCH_TOKEN`.

| Access on the Settings page | Scope | What the token can do |
| --- | --- | --- |
| Read | `products:read` | `pricewatch prices`, `pricewatch search`, `pricewatch auth status` |
| Read and add | `products:write` | also `pricewatch add` and `pricewatch remove` |

`pricewatch auth status` shows the account, the token and the address in
use. `pricewatch auth logout` removes the saved token; the token works until
it expires, or until you revoke it on the Settings page, which stops it at
once.

## Show the prices of your products

`pricewatch prices` lists your tracked products with their lowest price in
stock:

```console
$ pricewatch prices
PRODUCT           LOWEST    STORE      IN STOCK       CHECKED
Astro Bot PS5     18,75 €   Cultura    7 of 7 stores  5s ago
iPhone 13 128 Go  239,99 €  Cdiscount  6 of 6 stores  12s ago
```

`pricewatch prices PRODUCT` shows each store's price, stock, last check and
link. PRODUCT is any words of the product's name, in any order; case and
accents are ignored. A product ID, or its first 8 characters or more, also
works.

```console
$ pricewatch prices astro
Astro Bot PS5
New · lowest in stock 18,75 € at Cultura

STORE       PRICE    STOCK     CHECKED  LINK
Cultura     18,75 €  in stock  5s ago   https://www.cultura.com/p-astro-bot-10652697.html
Easycash    39,99 €  in stock  6s ago   https://bons-plans.easycash.fr/jeux-video/astro-bot-ps5-005263923
GameCash    43,47 €  in stock  6s ago   https://www.gamecash.fr/astro-bot-e109830.html
…
```

Stores are listed in stock first, cheapest first; then out of stock; then
the stores not checked yet ("checking…"); then paused ones. `--in-stock`
keeps the stores that have the product in stock. Prices are the latest the
stores showed: your Pricewatch plan sets how often they are checked (every
5 minutes on Free, every minute on Premium, every 5 seconds on Ultimate).

## Search the catalog

`pricewatch search QUERY` lists the products of Pricewatch's catalog whose
name has every word of QUERY, best match first, with the number of stores
that sell each one; ✓ marks the products you track. `--type` keeps one
product type (`video_game`, `console`, `smartphone`, `laptop`, `tcg`,
`other`), `--store` keeps the products one store sells and adds that
store's link (its name, such as `"E.Leclerc"`, or its code name, such as
`leclerc`), `--limit` sets how many are listed (20 by default, 100 at
most). Without QUERY, it lists the catalog.

```console
$ pricewatch search "iphone 13 pro"
PRODUCT                   TYPE   STORES  TRACKED  ID
iPhone 13 Pro 128 Go      Phone  5                28dfbce9-bd80-4280-a37b-1806a26a1efd
iPhone 13 Pro 256 Go      Phone  6                ff1862ef-cf05-4f44-948f-d8694e54e921
iPhone 13 Pro Max 128 Go  Phone  5                064b9333-1178-4060-a68d-0faf97da81de
iPhone 13 Pro Max 256 Go  Phone  6                52299ad9-cc00-426e-96bc-784796154caf
```

`pricewatch add --id ID` tracks a result.

## Add a product

```sh
pricewatch add "Elden Ring PS5"
pricewatch add "iPhone 13 128 Go" --condition any --max-price 449,90
pricewatch add "Astro Bot PS5" --wait
```

`pricewatch add NAME` first looks for NAME in Pricewatch's catalog:

- A product named exactly NAME (case, accents and spaces ignored) is
  tracked.
- When several products match, pricewatch asks which one. Without a
  terminal, it lists them with their IDs and stops with exit code 2: run it
  again with the exact name, `--id ID`, or `--yes` to take the first.
- When no product matches, Pricewatch searches every store for NAME and
  adds what it finds to the catalog, which takes a few seconds; then the
  product is tracked. `--new` does this even when the catalog has products
  that match. `--type` gives the product type (`video_game`, `console`,
  `smartphone`, `laptop`, `tcg`, `other`); without it, Pricewatch guesses it
  from NAME.

| Option | What it does |
| --- | --- |
| `--condition new\|used\|any` | which offers count: new (default), used, or both |
| `--max-price 450` | alerts only when a store's price is at or below 450 € |
| `--wait` | waits for the first price of each store, and shows them |
| `--json` | prints the tracked product as JSON |

Your plan limits how many products you track: the Free plan tracks 1.

## Stop tracking a product

```sh
pricewatch remove "elden ring ps5"
pricewatch remove 0b6f2c1e --yes
```

`pricewatch remove PRODUCT` (also `rm` or `untrack`) stops tracking one of
your products and deletes its price history. PRODUCT works as for `prices`:
words of the name, or the ID or its first 8 characters or more. When several
of your products match, pricewatch lists them and stops (exit code 2), unless
one is named exactly PRODUCT. It asks before removing; without a terminal,
`--yes` confirms.

## Settings: address, token, configuration file

Each setting comes from the first of these that is set:

| Setting | 1. Flag | 2. Environment variable | 3. Configuration file | 4. Default |
| --- | --- | --- | --- | --- |
| Pricewatch address | `--url` | `PRICEWATCH_URL` | `url` | `https://pricewatch.exe.xyz` |
| API token | `--token` | `PRICEWATCH_TOKEN` | `token` | none |

The configuration file is `pricewatch/config.json` in the user configuration
folder (`~/.config` on Linux, `~/Library/Application Support` on macOS,
`%AppData%` on Windows), or the file named by `PRICEWATCH_CONFIG`. On
Linux and macOS, pricewatch writes it readable by your user only (mode
600); on Windows, the permissions of your user folder protect it.

[CLAUDE RECOMMENDED – based on other users of a computer being able to see
the command lines of running programs] Give the token with
`pricewatch auth login` or `PRICEWATCH_TOKEN` rather than `--token`.

pricewatch reaches Pricewatch at its website address, and sends the token
in the `X-Pricewatch-Token` header. The site must answer without a sign-in
in front of it; otherwise pricewatch stops with "the server redirected the
request to …" or "… refused the request before it reached Pricewatch".

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | The request failed: network, server, or an unexpected answer |
| 2 | The command line is wrong, or a name matches several products |
| 3 | Nothing matches: no tracked product, or no store sells the product |
| 4 | Refused: not signed in; token unknown, expired or revoked, or without the scope; plan limit |

## JSON output

`prices`, `add` and `auth status` print JSON with `--json`, as Pricewatch's
API returns it. Prices are in cents (`priceCents`); the API adds fields over
time and does not remove them. The API is described in the
[Pricewatch README](https://github.com/DharmaBytesX/pricewatch#api-tokens-and-the-clis-api-apiv1).

```sh
pricewatch prices --json | jq -r '.[] | .name'
```

## License

Proprietary. See [LICENSE](LICENSE).
