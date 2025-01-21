package main

import (
	"log"

	"jihulab.com/jihulab/ultrafox/ultrafox/cmd/ultrafox/cmd"

	"github.com/spf13/cobra/doc"
)

func main() {
	ultrafox := cmd.RootCmd
	err := doc.GenMarkdownTree(ultrafox, "./docs/cli/")
	if err != nil {
		log.Fatal(err)
	}
}
