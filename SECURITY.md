# Security

## Reporting a problem

To report a security problem in this CLI or in the Pricewatch service, write
to simon.aurascussel@proton.me. Please do not open a public issue for it.
Say what you found and how to reproduce it; you will get an answer within a
few days.

Testing the service itself must stay within its
[terms of service](https://pricewatch.exe.xyz/terms#use): use your own
account and API tokens, and do not send large numbers of requests.

## How the CLI handles your API token

- `pricewatch auth login` saves the token in the configuration file
  (`pricewatch auth status` shows where), readable by your user only.
  `PRICEWATCH_TOKEN` or `--token` can give it instead; a token given on the
  command line can end up in your shell's history.
- The token is sent only to the Pricewatch address you use, in the
  `X-Pricewatch-Token` header, over HTTPS. The CLI refuses to send it to a
  plain `http://` address, except one on your own computer, and it never
  follows a redirect to another address.
- Text that comes from the server (product and store names, messages) is
  printed without the control characters that could drive your terminal.
- A token has read or write access and an expiry date. Revoke it on the
  Settings page of Pricewatch when you no longer need it; deleting your
  account revokes all of them.
