## pricewatch auth logout

Remove the saved token from the configuration file

### Synopsis

Removes the saved token from the configuration file. The token keeps working
until it expires: revoke it on Pricewatch's Settings page to stop it at once.

```
pricewatch auth logout [flags]
```

### Options

```
  -h, --help   help for logout
```

### Options inherited from parent commands

```
      --token string   API token (default: $PRICEWATCH_TOKEN or the config file); other users of the computer can see a flag, prefer the variable
      --url string     Pricewatch address (default: $PRICEWATCH_URL, the config file, or https://pricewatch.exe.xyz)
```

### SEE ALSO

* [pricewatch auth](pricewatch_auth.md)	 - Sign in with an API token, and check which one is used

