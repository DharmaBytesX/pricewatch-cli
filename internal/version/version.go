// Package version holds the CLI's version. Release builds set it with
//
//	-ldflags "-X github.com/DharmaBytesX/pricewatch-cli/internal/version.Version=1.2.3
//	          -X github.com/DharmaBytesX/pricewatch-cli/internal/version.Commit=abc1234
//	          -X github.com/DharmaBytesX/pricewatch-cli/internal/version.Date=2026-10-07"
package version

import "fmt"

// Set at build time; a development build keeps the defaults.
var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

// String describes the build, e.g. "1.2.3 (abc1234, 2026-10-07)".
func String() string {
	switch {
	case Commit != "" && Date != "":
		return fmt.Sprintf("%s (%s, %s)", Version, Commit, Date)
	case Commit != "":
		return fmt.Sprintf("%s (%s)", Version, Commit)
	}
	return Version
}

// UserAgent identifies the CLI to the Pricewatch API.
func UserAgent() string { return "pricewatch-cli/" + Version }
