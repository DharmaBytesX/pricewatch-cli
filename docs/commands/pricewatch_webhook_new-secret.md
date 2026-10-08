## pricewatch webhook new-secret

Replace the webhook's signing secret

### Synopsis

Replaces the secret that signs the webhook's requests, and prints the new one
once. Pricewatch signs the next requests with the new secret: give it to
your service. It asks before replacing; without a terminal, add --yes.

```
pricewatch webhook new-secret [flags]
```

### Examples

```
  pricewatch webhook new-secret
  pricewatch webhook new-secret --yes --json | jq -r .secret > secret.txt
```

### Options

```
  -h, --help   help for new-secret
      --json   print the answer as JSON: {"secret": …}
  -y, --yes    replace without asking
```

### Options inherited from parent commands

```
      --token string   API token (default: $PRICEWATCH_TOKEN or the config file); other users of the computer can see a flag, prefer the variable
      --url string     Pricewatch address (default: $PRICEWATCH_URL, the config file, or https://pricewatch.exe.xyz)
```

### SEE ALSO

* [pricewatch webhook](pricewatch_webhook.md)	 - Send your alerts to your own service, and check what was sent

