package main

import (
	"fmt"
	"os"
)

func main() {
	// Recover from panics to show user-friendly error messages
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "\nArx encountered an unexpected error.\n")
			fmt.Fprintf(os.Stderr, "If this persists, please report it at: https://github.com/pauvalls/arx/issues\n")
			fmt.Fprintf(os.Stderr, "\nDebug info: %v\n", r)
			os.Exit(1)
		}
	}()

	if err := Execute(); err != nil {
		printError(err)
		os.Exit(1)
	}
}
