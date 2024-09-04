package serve

import "github.com/urfave/cli/v2"

func RootCommand() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Serving related commands",
		Subcommands: []*cli.Command{
			BookCommand(),
		},
	}
}
