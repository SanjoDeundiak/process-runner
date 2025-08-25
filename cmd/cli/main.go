package main

import (
	"fmt"
	"os"
)

func main() {
	root := NewRootCmd()

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
