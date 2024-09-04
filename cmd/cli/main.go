package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sort"

	"github.com/Bornholm/ai-adventure/internal/command"
	"github.com/urfave/cli/v2"
)

func main() {
	ctx := context.Background()

	app := &cli.App{
		Name:        "ai-adv",
		Description: "Generate CYOA books with AI",
		Commands:    command.Commands(),
		Before: func(ctx *cli.Context) error {
			workdir := ctx.String("workdir")
			if workdir != "" {
				if err := os.Chdir(workdir); err != nil {
					return errors.New("could not change working directory")
				}
			}

			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "workdir",
				Value: "",
				Usage: "The working directory",
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	app.ExitErrHandler = func(ctx *cli.Context, err error) {
		if err == nil {
			return
		}

		slog.Error("app exited with error", slog.Any("error", err))
		os.Exit(1)
	}

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		slog.Error("could not run app", slog.Any("error", err))
		os.Exit(1)
	}
}
