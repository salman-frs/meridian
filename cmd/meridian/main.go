package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/salman-frs/meridian/internal/app"
)

func main() {
	if err := app.NewRootCommand().Execute(); err != nil {
		if strings.TrimSpace(err.Error()) != "" {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(app.ExitCode(err))
	}
}
