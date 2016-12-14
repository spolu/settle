package main

import (
	"os"

	"github.com/spolu/settle/cli"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		cli.Help()
		return
	}
}
