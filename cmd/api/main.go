package main

import (
	"errors"
	"log"
	"os"
	_ "pasteAPI/docs"
	"pasteAPI/internal/app"
	"pasteAPI/internal/config"
)

// @title Paste API
// @version 1.0
// @description This is a Paste API server. It is used to publish, read, edit text posts.
// @host localhost:8080
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
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
