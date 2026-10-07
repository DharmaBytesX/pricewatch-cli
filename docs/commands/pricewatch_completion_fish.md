## pricewatch completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	pricewatch completion fish | source

To load completions for every new session, execute once:

	pricewatch completion fish > ~/.config/fish/completions/pricewatch.fish

You will need to start a new shell for this setup to take effect.


```
pricewatch completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --token string   API token (default: $PRICEWATCH_TOKEN or the config file); other users of the computer can see a flag, prefer the variable
      --url string     Pricewatch address (default: $PRICEWATCH_URL, the config file, or https://pricewatch.exe.xyz)
```

### SEE ALSO

* [pricewatch completion](pricewatch_completion.md)	 - Generate the autocompletion script for the specified shell

