## pricewatch auth login

Save an API token, after checking it with Pricewatch

```
pricewatch auth login [flags]
```

### Examples

```
  pricewatch auth login                          # paste the token when asked
  pricewatch auth login --with-token < token.txt # read it from standard input
```

### Options

```
  -h, --help         help for login
      --with-token   read the token from standard input
```

### Options inherited from parent commands

```
      --token string   API token (default: $PRICEWATCH_TOKEN or the config file); other users of the computer can see a flag, prefer the variable
      --url string     Pricewatch address (default: $PRICEWATCH_URL, the config file, or https://pricewatch.exe.xyz)
```

### SEE ALSO

* [pricewatch auth](pricewatch_auth.md)	 - Sign in with an API token, and check which one is used

