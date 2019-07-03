package main

import (
	"fmt"
	"os"

	"github.com/izumin5210/ghsync/cmd/ghsync/cmd"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	return cmd.New().Execute()
}
