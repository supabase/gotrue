package main

import (
	"log"

	"github.com/netlify/gotrue/cmd"
	_ "github.com/netlify/gotrue/docs"
)

func main() {
	if err := cmd.RootCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
