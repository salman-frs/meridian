package main

import (
	"context"
	"fmt"
	"os"

	e2eapp "github.com/salman-frs/meridian/examples/k3s-e2e/app"
)

func main() {
	if err := e2eapp.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "meridian-k3s-e2e: %v\n", err)
		os.Exit(e2eapp.ExitCode(err))
	}
}
