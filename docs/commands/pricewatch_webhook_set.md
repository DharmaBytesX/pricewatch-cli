## pricewatch webhook set

Send your alerts to this address

### Synopsis

Sends your alerts to URL, which must start with https:// and be reachable from
the internet. Pricewatch sends a POST request with a JSON body for each
alert.

For a new webhook, Pricewatch creates a signing secret and pricewatch prints
it once: Pricewatch signs every request with it (the webhook-signature
header), so your service can check that the request comes from Pricewatch.
Copy it then; Pricewatch does not show it again, and "pricewatch webhook
new-secret" replaces it. Changing the address keeps the secret.

In a script, --json prints the answer, and jq -r .secret gives the secret
alone.

```
pricewatch webhook set URL [flags]
```

### Examples

```
  pricewatch webhook set https://example.com/pricewatch
  pricewatch webhook set https://hooks.example.com/abc123 --json | jq -r .secret > secret.txt
```

### Options

```
  -h, --help   help for set
      --json   print the answer as JSON: {"url": …, "secret": …}; the secret only for a new webhook
```

### Options inherited from parent commands

```
      --token string   API token (default: $PRICEWATCH_TOKEN or the config file); other users of the computer can see a flag, prefer the variable
      --url string     Pricewatch address (default: $PRICEWATCH_URL, the config file, or https://pricewatch.exe.xyz)
```

### SEE ALSO

* [pricewatch webhook](pricewatch_webhook.md)	 - Send your alerts to your own service, and check what was sent

