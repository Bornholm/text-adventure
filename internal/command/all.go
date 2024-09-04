package command

import (
	"github.com/Bornholm/ai-adventure/internal/command/generate"
	"github.com/Bornholm/ai-adventure/internal/command/serve"
	"github.com/urfave/cli/v2"
)

func Commands() []*cli.Command {
	return []*cli.Command{
		generate.RootCommand(),
		serve.RootCommand(),
	}
}
