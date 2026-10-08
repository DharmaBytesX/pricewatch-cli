## pricewatch webhook remove

Stop sending alerts to the webhook

### Synopsis

Removes the webhook and its signing secret: Pricewatch stops sending alerts to
it. It asks before removing; without a terminal, add --yes.

```
pricewatch webhook remove [flags]
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

* [pricewatch webhook](pricewatch_webhook.md)	 - Send your alerts to your own service, and check what was sent

