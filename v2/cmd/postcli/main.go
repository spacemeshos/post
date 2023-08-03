package main

import "github.com/spacemeshos/post/v2/cmd/postcli/cmd"

var (
	// Version is the version of the binary.
	Version = "0.0.0"

	// Commit is the commit hash of the binary.
	Commit = ""
)

func main() {
	cmd.Version = Version
	cmd.Commit = Commit
	cmd.Execute()
}
