package main

import (
	"errors"
	_ "github.com/zhukovrost/pasteAPI/docs"
	"github.com/zhukovrost/pasteAPI/internal/app"
	"github.com/zhukovrost/pasteAPI/internal/config"
	"log"
	"os"
)

// @title Paste API
// @version 1.0
// @description This is a Paste API server. It is used to publish, read, edit text posts.
// @host localhost:8080
// @schemes http

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description To authorize, use the format: Bearer <your_token>. For example: Bearer abc123xyz

// @security BearerAuth
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
