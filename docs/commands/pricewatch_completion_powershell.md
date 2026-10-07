## pricewatch completion powershell

Generate the autocompletion script for powershell

### Synopsis

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	pricewatch completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```
pricewatch completion powershell [flags]
```

### Options

```
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --token string   API token (default: $PRICEWATCH_TOKEN or the config file); other users of the computer can see a flag, prefer the variable
      --url string     Pricewatch address (default: $PRICEWATCH_URL, the config file, or https://pricewatch.exe.xyz)
```

### SEE ALSO

* [pricewatch completion](pricewatch_completion.md)	 - Generate the autocompletion script for the specified shell

