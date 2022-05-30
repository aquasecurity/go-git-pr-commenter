package app

import (
	"github.com/urfave/cli/v2"
)

func NewApp() *cli.App {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Commands = []*cli.Command{
		{
			Name:   "cmd",
			Action: Action,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "file",
					Aliases: []string{"f"},
					Usage:   "Target file",
				},
				&cli.StringFlag{
					Name:    "comment",
					Aliases: []string{"c"},
					Usage:   "PR comment",
				},
				&cli.StringFlag{
					Name:    "vendor",
					Aliases: []string{"v"},
					Usage:   "The vendor for the comment mock|github|bitbucket",
				},
				&cli.IntFlag{
					Name:    "start-line",
					Aliases: []string{"s"},
					Usage:   "Comment start line",
				},
				&cli.IntFlag{
					Name:    "end-line",
					Aliases: []string{"e"},
					Usage:   "Comment end line",
				},
				&cli.StringFlag{
					Name:  "repo",
					Usage: "The repo name",
				},
				&cli.IntFlag{
					Name:  "pr-number",
					Usage: "The pr number",
				},
				&cli.StringFlag{
					Name:  "owner",
					Usage: "The repo owner",
				},
			},
		},
	}

	return app
}
