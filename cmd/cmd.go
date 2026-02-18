package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"
	"github.com/webitel/im-providers-service/config"
)

func Run() error {
	app := &cli.App{
		Name:  "im-providers-service",
		Usage: "IM Providers Service",
		Commands: []*cli.Command{
			apiCmd(),
		},
	}

	return app.Run(os.Args)
}

func apiCmd() *cli.Command {
	return &cli.Command{
		Name:    "server",
		Aliases: []string{"s"},
		Usage:   "Run the gRPC server",
		Action: func(c *cli.Context) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			app := NewApp(cfg)

			if err := app.Start(c.Context); err != nil {
				return err
			}

			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
			<-stop

			slog.Info("Shutting down...")
			return app.Stop(context.Background())
		},
	}
}
