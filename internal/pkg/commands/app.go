package commands

import (
	"github.com/aquasecurity/go-git-pr-commenter/internal/app/commenter"
	"github.com/urfave/cli/v2"
)

func NewApp() *cli.App {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Commands = []*cli.Command{
		{
			Name:   "cmd",
			Action: commenter.Action,
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
				&cli.StringFlag{
					Name:    "start-line",
					Aliases: []string{"s"},
					Usage:   "Comment start line",
				},
				&cli.StringFlag{
					Name:    "end-line",
					Aliases: []string{"e"},
					Usage:   "Comment end line",
				},
			},
		},
	}

	return app
}
