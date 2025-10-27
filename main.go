package main

import (
	"os"

	"github.com/pixfid/luft/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
