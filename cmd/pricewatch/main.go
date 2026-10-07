// Command pricewatch shows the prices of the products you track on
// Pricewatch, and adds products to track. See README.md.
package main

import (
	"os"

	"github.com/DharmaBytesX/pricewatch-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute(cli.NewApp(), os.Args[1:]))
}
