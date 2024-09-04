package generate

import "github.com/urfave/cli/v2"

func RootCommand() *cli.Command {
	return &cli.Command{
		Name:  "generate",
		Usage: "Generation related commands",
		Subcommands: []*cli.Command{
			BookCommand(),
		},
	}
}
