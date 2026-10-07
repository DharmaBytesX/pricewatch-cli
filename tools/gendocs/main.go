// Command gendocs writes the command reference (docs/commands) from the
// commands' own help. Run "make docs" after changing a command; CI fails
// when the reference does not match the commands.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra/doc"

	"github.com/DharmaBytesX/pricewatch-cli/internal/cli"
)

func main() {
	dir := "docs/commands"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	root := cli.NewRootCommand(cli.NewApp())
	root.DisableAutoGenTag = true // no date: the files change only when the help does
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
	// Start from an empty directory, so a removed command loses its page.
	if err := os.RemoveAll(dir); err != nil {
		fail(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fail(err)
	}
	if err := doc.GenMarkdownTree(root, dir); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "gendocs:", err)
	os.Exit(1)
}
