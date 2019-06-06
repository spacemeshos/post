package main

import (
	"fmt"
	"github.com/spacemeshos/post/cmd/server"
	"os"
)

func main() {
	if err := server.Cmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
