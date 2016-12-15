package main

import (
	"os"

	"github.com/spolu/settle/cli"
	"github.com/spolu/settle/lib/out"
)

func main() {
	cli, err := cli.New(os.Args[1:])
	if err != nil {
		out.Errof("Error: %s", err.Error())
		os.Exit(1)
	}

	err = cli.Run()
	if err != nil {
		out.Errof("Error: %s", err.Error())
		os.Exit(1)
	}
}
