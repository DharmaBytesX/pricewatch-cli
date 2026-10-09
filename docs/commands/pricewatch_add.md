## pricewatch add

Track a product: from the catalog, or found in every store

### Synopsis

Tracks a product, the way the website's "Track a product" panel does.

pricewatch first looks for NAME in Pricewatch's catalog:
  - a product with exactly this name is tracked;
  - when several products match, it asks which one (or, without a terminal,
    lists them and stops: give the exact name, --id, or --yes);
  - when none matches, Pricewatch searches every store for NAME, adds what
    it finds to the catalog, and the product is tracked. This takes a few
    seconds.

The first prices arrive a few seconds after the product is tracked; --wait
shows them. Your plan limits how many products you track.

```
pricewatch add NAME [flags]
```

### Examples

```
  pricewatch add "Elden Ring PS5"
  pricewatch add "iPhone 13 128 Go" --condition any --target-price 450
  pricewatch add "Mario Kart World" --type video_game --wait
  pricewatch add --id 0b6f2c1e-1c4b-4f1e-9a51-7d3c2a1b9e00
```

### Options

```
      --condition string      new, used, or any (used & new) (default "new")
  -h, --help                  help for add
      --id string             track this catalog product ID instead of searching by NAME
      --json                  print the tracked product as JSON
      --new                   search the stores for NAME even when the catalog has products that match
      --target-price string   alert only at or below this price in euros, e.g. 450 or 449,90
      --timeout duration      how long to wait for the store search and for --wait (default 2m0s)
      --type string           type of a product the stores are searched for: video_game, console, smartphone, laptop, vr_headset, pc_component, tv, tcg, other (default: guessed from NAME)
      --wait                  wait for the first prices and show them
      --yes                   when several catalog products match, take the best match without asking
```

### Options inherited from parent commands

```
      --token string   API token (default: $PRICEWATCH_TOKEN or the config file); other users of the computer can see a flag, prefer the variable
      --url string     Pricewatch address (default: $PRICEWATCH_URL, the config file, or https://pricewatch.exe.xyz)
```

### SEE ALSO

* [pricewatch](pricewatch.md)	 - Follow product prices and stock in French stores

