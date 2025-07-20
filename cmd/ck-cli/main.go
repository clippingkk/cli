package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/clippingkk/cli/internal/commands"
	"github.com/urfave/cli/v2"
)

var (
	// Version is set at build time
	Version = "dev"
	// Commit is set at build time  
	Commit = "unknown"
)

func main() {
	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interruption signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		cancel()
	}()

	app := &cli.App{
		Name:        "ck-cli",
		Usage:       "Parse Amazon Kindle clippings and sync to ClippingKK service",
		Version:     fmt.Sprintf("%s (%s)", Version, Commit),
		Description: "ClippingKK CLI tool for parsing Kindle's My Clippings.txt file into structured JSON format and syncing with ClippingKK web service.",
		Authors: []*cli.Author{
			{
				Name:  "Annatar He",
				Email: "annatar.he+ck.cli@gmail.com",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to configuration file",
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "token",
				Aliases: []string{"t"},
				Usage:   "Authentication token for ClippingKK service",
				Value:   "",
			},
		},
		Commands: []*cli.Command{
			commands.LoginCommand,
			commands.ParseCommand,
		},
		Before: func(c *cli.Context) error {
			// Inject global configuration context
			commands.SetContext(ctx)
			return nil
		},
	}

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}