## pricewatch webhook test

Send a test event to the webhook now

### Synopsis

Sends a test event to the webhook now, and shows your service's answer. A test
event has the same fields as an alert, with example values, and the type
"test". The exit code is 1 when the event was not delivered.

```
pricewatch webhook test [flags]
```

### Options

```
  -h, --help   help for test
      --json   print the delivery as JSON: {"delivery": {…}}
```

### Options inherited from parent commands

```
      --token string   API token (default: $PRICEWATCH_TOKEN or the config file); other users of the computer can see a flag, prefer the variable
      --url string     Pricewatch address (default: $PRICEWATCH_URL, the config file, or https://pricewatch.exe.xyz)
```

### SEE ALSO

* [pricewatch webhook](pricewatch_webhook.md)	 - Send your alerts to your own service, and check what was sent

