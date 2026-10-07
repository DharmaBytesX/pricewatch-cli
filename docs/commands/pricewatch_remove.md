## pricewatch remove

Stop tracking a product and delete its price history

### Synopsis

Stops tracking one of your products and deletes its price history. PRODUCT is
any words of the product's name, in any order (case and accents ignored), or
its ID or the first 8 characters of it or more.

When several of your products match, pricewatch lists them and stops: give
the exact name or the ID. It asks before removing; without a terminal, add
--yes.

```
pricewatch remove PRODUCT [flags]
```

### Examples

```
  pricewatch remove "elden ring ps5"
  pricewatch remove 0b6f2c1e --yes
```

### Options

```
  -h, --help   help for remove
  -y, --yes    remove without asking
```

### Options inherited from parent commands

```
      --token string   API token (default: $PRICEWATCH_TOKEN or the config file); other users of the computer can see a flag, prefer the variable
      --url string     Pricewatch address (default: $PRICEWATCH_URL, the config file, or https://pricewatch.exe.xyz)
```

### SEE ALSO

* [pricewatch](pricewatch.md)	 - Follow product prices and stock in French stores

