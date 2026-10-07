## pricewatch prices

Show the prices and stock of your tracked products

### Synopsis

Without PRODUCT, lists your tracked products with their lowest price in stock.

With PRODUCT, shows each store's price, stock, last check and link for the
tracked products whose name has every word of PRODUCT (case and accents
ignored), or whose ID starts with PRODUCT.

Prices are the latest the stores showed: your plan sets how often they are
checked.

```
pricewatch prices [PRODUCT] [flags]
```

### Examples

```
  pricewatch prices
  pricewatch prices "iphone 13 128"
  pricewatch prices elden ring --in-stock
  pricewatch prices --json | jq '.[].name'
```

### Options

```
  -h, --help       help for prices
      --in-stock   show only the stores that have the product in stock
      --json       print the products as JSON, as the API returns them
```

### Options inherited from parent commands

```
      --token string   API token (default: $PRICEWATCH_TOKEN or the config file); other users of the computer can see a flag, prefer the variable
      --url string     Pricewatch address (default: $PRICEWATCH_URL, the config file, or https://pricewatch.exe.xyz)
```

### SEE ALSO

* [pricewatch](pricewatch.md)	 - Follow product prices and stock in French stores

