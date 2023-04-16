package github

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v44/github"
	"golang.org/x/oauth2"
)

const githubAbuseErrorRetries = 6

type connector struct {
	prs      *github.PullRequestsService
	comments *github.IssuesService
	owner    string
	repo     string
	prNumber int
}

type existingComment struct {
	filename  *string
	comment   *string
	commentId *int64
}

type commentFn func() (*github.Response, error)

// create github connector and check if supplied pr number exists
func createConnector(apiUrl, token, owner, repo string, prNumber int, isEnterprise bool) (*connector, error) {

	client, err := newGithubClient(apiUrl, token, isEnterprise)
	if err != nil {
		return nil, err
	}
	if _, _, err := client.PullRequests.Get(context.Background(), owner, repo, prNumber); err != nil {
		return nil, newPrDoesNotExistError(owner, repo, prNumber)
	}

	return &connector{
		prs:      client.PullRequests,
		comments: client.Issues,
		owner:    owner,
		repo:     repo,
		prNumber: prNumber,
	}, nil
}

func newGithubClient(apiUrl, token string, isEnterprise bool) (*github.Client, error) {

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	if isEnterprise {
		return github.NewEnterpriseClient(apiUrl, apiUrl, tc)
	}

	return github.NewClient(tc), nil
}

func (c *connector) writeReviewComment(block *github.PullRequestComment, commentId *int64) error {

	ctx := context.Background()
	if commentId != nil {
		return writeCommentWithRetries(c.owner, c.repo, c.prNumber, func() (*github.Response, error) {
			_, resp, err := c.prs.EditComment(ctx, c.owner, c.repo, *commentId, &github.PullRequestComment{
				Body: block.Body,
			})
			return resp, err
		})
	}

	return writeCommentWithRetries(c.owner, c.repo, c.prNumber, func() (*github.Response, error) {
		_, resp, err := c.prs.CreateComment(ctx, c.owner, c.repo, c.prNumber, block)
		return resp, err
	})
}

func writeCommentWithRetries(owner, repo string, prNumber int, commentFn commentFn) error {

	var abuseError AbuseRateLimitError
	for i := 0; i < githubAbuseErrorRetries; i++ {

		retrySeconds := i * i
		time.Sleep(time.Second * time.Duration(retrySeconds))

		if resp, err := commentFn(); err != nil {
			// If we get a 403 or 422, we are being rate or abuse limited by GitHub,
			// and we want to retry, while increasing the wait time between retries.
			if resp != nil && (resp.StatusCode == 422 || resp.StatusCode == 403) {
				abuseError = newAbuseRateLimitError(owner, repo, prNumber, retrySeconds)
				continue
			}
			return fmt.Errorf("write comment: %v", err)
		}
		return nil
	}
	return abuseError
}

func (c *connector) getFilesForPr() ([]*github.CommitFile, error) {

	files, _, err := c.prs.ListFiles(context.Background(), c.owner, c.repo, c.prNumber, nil)
	if err != nil {
		return nil, err
	}

	var commitFiles []*github.CommitFile
	for _, file := range files {
		if *file.Status != "deleted" {
			commitFiles = append(commitFiles, file)
		}
	}
	return commitFiles, nil
}

func (c *connector) getExistingComments() ([]*existingComment, error) {

	ctx := context.Background()
	comments, _, err := c.prs.ListComments(ctx, c.owner, c.repo, c.prNumber, &github.PullRequestListCommentsOptions{})
	if err != nil {
		return nil, err
	}

	var existingComments []*existingComment
	for _, comment := range comments {
		existingComments = append(existingComments, &existingComment{
			filename:  comment.Path,
			comment:   comment.Body,
			commentId: comment.ID,
		})
	}
	return existingComments, nil
}
