package commenter

import (
	"fmt"
	"github.com/aquasecurity/go-git-pr-commenter/internal/pkg/commenter"
	"github.com/aquasecurity/go-git-pr-commenter/internal/pkg/commenter/github"
	"github.com/aquasecurity/go-git-pr-commenter/internal/pkg/commenter/mock"
	"github.com/urfave/cli/v2"
)

func Action(ctx *cli.Context) (err error) {
	var c = commenter.Repository(nil)
	switch ctx.String("vendor") {
	case "mock":
		c = commenter.Repository(mock.NewMock())
	case "github":
		c = commenter.Repository(github.NewGithub())
	}

	err = c.WriteMultiLineComment(
		ctx.String("file"),
		ctx.String("comment"),
		ctx.String("start-line"),
		ctx.String("end-line"))
	if err != nil {
		return fmt.Errorf("failed write comment: %w", err)
	}

	return nil

}
