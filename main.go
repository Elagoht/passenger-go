package main

import (
	"os"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/legacy"
)

func main() {
	// migrate runs before collage so it needs neither CSRF_KEY nor the new DB.
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		os.Exit(legacy.RunCLI(os.Args[2:], os.Stdout, os.Stderr))
	}
	collage.Main(newApp)
}
