## pricewatch completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(pricewatch completion bash)

To load completions for every new session, execute once:

#### Linux:

	pricewatch completion bash > /etc/bash_completion.d/pricewatch

#### macOS:

	pricewatch completion bash > $(brew --prefix)/etc/bash_completion.d/pricewatch

You will need to start a new shell for this setup to take effect.


```
pricewatch completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --token string   API token (default: $PRICEWATCH_TOKEN or the config file); other users of the computer can see a flag, prefer the variable
      --url string     Pricewatch address (default: $PRICEWATCH_URL, the config file, or https://pricewatch.exe.xyz)
```

### SEE ALSO

* [pricewatch completion](pricewatch_completion.md)	 - Generate the autocompletion script for the specified shell

