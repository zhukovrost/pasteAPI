package main

import (
	"errors"
	"log"
	"os"
	"pasteAPI/internal/app"
	"pasteAPI/internal/config"
)

func main() {
	cfg, err := config.New()

	if err != nil {
		switch {
		case errors.Is(err, config.ErrDisplayAndExit):
			log.Printf("Version:\t%s\n", config.Version)
			log.Printf("Build time:\t%s\n", config.BuildTime)
			os.Exit(0)
		default:
			log.Printf("Error processing env config: %v\n", err)
			os.Exit(1)
		}
	}

	app.Run(cfg)
}
