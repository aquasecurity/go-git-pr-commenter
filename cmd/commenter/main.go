package main

import (
	"github.com/aquasecurity/go-git-pr-commenter/internal/pkg/commands"
	"log"
	"os"
)

func main() {
	app := commands.NewApp()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
