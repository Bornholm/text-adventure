package command

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func Version() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Show version",
		Action: func(ctx *cli.Context) error {
			projectVersion := ctx.String("projectVersion")
			gitRef := ctx.String("gitRef")
			buildDate := ctx.String("buildDate")

			fmt.Printf("%s (%s) - %s\n", projectVersion, gitRef, buildDate)

			os.Exit(0)

			return nil
		},
	}
}
