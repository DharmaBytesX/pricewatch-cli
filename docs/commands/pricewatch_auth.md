## pricewatch auth

Sign in with an API token, and check which one is used

### Synopsis

Create an API token on Pricewatch's Settings page ("API tokens"): choose its
access ("Read", or "Read and add"; "Manage the webhook" for the webhook
commands) and when it expires. "pricewatch auth login" saves it in the
configuration file.

The token is taken from, in order: the --token flag, the PRICEWATCH_TOKEN
variable, then the configuration file.

### Options

```
  -h, --help   help for auth
```

### Options inherited from parent commands

```
      --token string   API token (default: $PRICEWATCH_TOKEN or the config file); other users of the computer can see a flag, prefer the variable
      --url string     Pricewatch address (default: $PRICEWATCH_URL, the config file, or https://pricewatch.exe.xyz)
```

### SEE ALSO

* [pricewatch](pricewatch.md)	 - Follow product prices and stock in French stores
* [pricewatch auth login](pricewatch_auth_login.md)	 - Save an API token, after checking it with Pricewatch
* [pricewatch auth logout](pricewatch_auth_logout.md)	 - Remove the saved token from the configuration file
* [pricewatch auth status](pricewatch_auth_status.md)	 - Show the account, the token and the address in use

