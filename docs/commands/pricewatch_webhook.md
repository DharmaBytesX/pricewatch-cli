## pricewatch webhook

Send your alerts to your own service, and check what was sent

### Synopsis

Pricewatch can send each of your alerts (a price drop, a product back in
stock) to an HTTPS address of yours, as a signed JSON request, in addition
to Discord. Your service, or an automation tool that receives webhooks,
then acts on it.

Without a command, shows the webhook's address and its 10 latest
deliveries: the event, whether it was sent, the answer of your service
(its HTTP status, or why there was none), the number of attempts, and when.

Showing the webhook needs an API token with "Read" access; setting,
testing or removing it needs "Write" access. The event body and how to
check the signature are described on Pricewatch's docs page, at
/docs#webhooks.

```
pricewatch webhook [flags]
```

### Examples

```
  pricewatch webhook set https://example.com/pricewatch
  pricewatch webhook test
  pricewatch webhook
  pricewatch webhook --json | jq '.deliveries[] | select(.status == "failed")'
```

### Options

```
  -h, --help   help for webhook
      --json   print the address and the deliveries as JSON: {"url": …, "deliveries": […]}
```

### Options inherited from parent commands

```
      --token string   API token (default: $PRICEWATCH_TOKEN or the config file); other users of the computer can see a flag, prefer the variable
      --url string     Pricewatch address (default: $PRICEWATCH_URL, the config file, or https://pricewatch.exe.xyz)
```

### SEE ALSO

* [pricewatch](pricewatch.md)	 - Follow product prices and stock in French stores
* [pricewatch webhook new-secret](pricewatch_webhook_new-secret.md)	 - Replace the webhook's signing secret
* [pricewatch webhook remove](pricewatch_webhook_remove.md)	 - Stop sending alerts to the webhook
* [pricewatch webhook set](pricewatch_webhook_set.md)	 - Send your alerts to this address
* [pricewatch webhook test](pricewatch_webhook_test.md)	 - Send a test event to the webhook now

