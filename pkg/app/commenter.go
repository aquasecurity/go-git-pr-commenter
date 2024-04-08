package app

import (
	"fmt"
	"os"

	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter"
	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/azure"
	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/bitbucket"
	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/github"
	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/gitlab"
	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/mock"
	"github.com/urfave/cli/v2"
)

func Action(ctx *cli.Context) (err error) {
	var c = commenter.Repository(nil)
	switch ctx.String("vendor") {
	case "mock":
		c = commenter.Repository(mock.NewMock())
	case "github":
		token := os.Getenv("GITHUB_TOKEN")
		r, err := github.NewGithub(token, ctx.String("owner"), ctx.String("repo"), ctx.Int("pr-number"))
		if err != nil {
			return err
		}
		c = commenter.Repository(r)
	case "gitlab":
		token := os.Getenv("GITLAB_TOKEN")
		r, err := gitlab.NewGitlab(
			token)
		if err != nil {
			return err
		}
		c = commenter.Repository(r)
	case "azure":
		token := os.Getenv("AZURE_TOKEN")
		r, err := azure.NewAzure(token, ctx.String("project"), ctx.String("collection-url"), ctx.String("repo-id"), ctx.String("pr-number"))
		if err != nil {
			return err
		}
		c = commenter.Repository(r)
	case "bitbucket":
		userName := os.Getenv("BITBUCKET_USER")
		token := os.Getenv("BITBUCKET_TOKEN")
		r, err := bitbucket.NewBitbucket(userName, token)
		if err != nil {
			return err
		}
		c = commenter.Repository(r)
	}

	err = c.WriteMultiLineComment(
		ctx.String("file"),
		ctx.String("comment"),
		ctx.Int("start-line"),
		ctx.Int("end-line"))
	if err != nil {
		return fmt.Errorf("failed write comment: %w", err)
	}

	return nil

}
