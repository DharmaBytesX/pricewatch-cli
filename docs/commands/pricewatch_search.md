## pricewatch search

Search Pricewatch's catalog

### Synopsis

Lists the catalog products whose name has every word of QUERY (case and
accents ignored), best match first, with the number of stores that sell each
one. ✓ marks the products you track. Without QUERY, lists the catalog.

--store keeps the products one store sells, and shows that store's link:
give its name ("E.Leclerc", "Amazon FR") or its code name ("leclerc",
"amazon").

Track a result with: pricewatch add --id ID

```
pricewatch search [QUERY] [flags]
```

### Examples

```
  pricewatch search "elden ring"
  pricewatch search iphone --type smartphone --limit 50
  pricewatch search --type tcg
  pricewatch search --store leclerc --limit 100
```

### Options

```
  -h, --help           help for search
      --json           print the products as JSON, as the API returns them
      --limit int      how many products to list, 1 to 100 (default 20)
      --store string   only the products this store sells, e.g. "E.Leclerc" or leclerc
      --type string    only this product type: video_game, console, smartphone, laptop, vr_headset, pc_component, tv, tcg, other
```

### Options inherited from parent commands

```
      --token string   API token (default: $PRICEWATCH_TOKEN or the config file); other users of the computer can see a flag, prefer the variable
      --url string     Pricewatch address (default: $PRICEWATCH_URL, the config file, or https://pricewatch.exe.xyz)
```

### SEE ALSO

* [pricewatch](pricewatch.md)	 - Follow product prices and stock in French stores

