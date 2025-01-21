package main

import (
	"fmt"
	"log"
	"path"
	"path/filepath"
	"strings"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/cmd/ultrafox/cmd"

	"github.com/spf13/cobra/doc"
)

func main() {
	ultrafox := cmd.RootCmd
	const fmTemplate = `---
date: %s
title: "%s"
slug: %s
url: %s
---
`

	filePrepender := func(filename string) string {
		now := time.Now().Format(time.RFC3339)
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))
		url := "/cli/" + strings.ToLower(base) + "/"
		return fmt.Sprintf(fmTemplate, now, strings.Replace(base, "_", " ", -1), base, url)
	}
	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		return "/cli/" + strings.ToLower(base) + "/"
	}
	//err := doc.GenMarkdownTree(ultrafox, "./docs/cli/")
	err := doc.GenMarkdownTreeCustom(ultrafox, "./hugo/", filePrepender, linkHandler)
	if err != nil {
		log.Fatal(err)
	}
}
