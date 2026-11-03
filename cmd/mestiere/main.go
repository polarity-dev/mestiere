package main

import (
	"os"

	"github.com/polarity-dev/mestiere/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
