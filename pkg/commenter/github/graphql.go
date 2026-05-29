package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Default endpoint for github.com. GHE rewrites this; we only support github.com today.
const defaultGraphQLEndpoint = "https://api.github.com/graphql"

const reviewThreadsQuery = `
query($owner: String!, $name: String!, $number: Int!, $threadCursor: String) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviewThreads(first: 100, after: $threadCursor) {
        pageInfo { hasNextPage endCursor }
        nodes {
          id
          isResolved
          isOutdated
          path
          line
          startLine
          comments(first: 100) {
            nodes {
              databaseId
              body
              path
              line
              startLine
            }
          }
        }
      }
    }
  }
}`

type gqlReviewComment struct {
	DatabaseID int64   `json:"databaseId"`
	Body       string  `json:"body"`
	Path       string  `json:"path"`
	Line       *int    `json:"line"`
	StartLine  *int    `json:"startLine"`
}

type gqlReviewThread struct {
	ID         string `json:"id"`
	IsResolved bool   `json:"isResolved"`
	IsOutdated bool   `json:"isOutdated"`
	Path       string `json:"path"`
	Line       *int   `json:"line"`
	StartLine  *int   `json:"startLine"`
	Comments   struct {
		Nodes []gqlReviewComment `json:"nodes"`
	} `json:"comments"`
}

type gqlPageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type gqlReviewThreadsResponse struct {
	Data struct {
		Repository struct {
			PullRequest struct {
				ReviewThreads struct {
					PageInfo gqlPageInfo       `json:"pageInfo"`
					Nodes    []gqlReviewThread `json:"nodes"`
				} `json:"reviewThreads"`
			} `json:"pullRequest"`
		} `json:"repository"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// fetchReviewThreads pages through every review thread on the PR and returns them flattened.
func (c *connector) fetchReviewThreads(ctx context.Context, token, endpoint string) ([]gqlReviewThread, error) {
	if endpoint == "" {
		endpoint = defaultGraphQLEndpoint
	}

	var all []gqlReviewThread
	cursor := ""
	for {
		vars := map[string]interface{}{
			"owner":  c.owner,
			"name":   c.repo,
			"number": c.prNumber,
		}
		if cursor != "" {
			vars["threadCursor"] = cursor
		} else {
			vars["threadCursor"] = nil
		}

		body, err := json.Marshal(map[string]interface{}{
			"query":     reviewThreadsQuery,
			"variables": vars,
		})
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("graphql request: %w", err)
		}
		raw, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode/100 != 2 {
			return nil, fmt.Errorf("graphql http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
		}

		var parsed gqlReviewThreadsResponse
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, fmt.Errorf("graphql decode: %w", err)
		}
		if len(parsed.Errors) > 0 {
			msgs := make([]string, 0, len(parsed.Errors))
			for _, e := range parsed.Errors {
				msgs = append(msgs, e.Message)
			}
			return nil, fmt.Errorf("graphql: %s", strings.Join(msgs, "; "))
		}

		page := parsed.Data.Repository.PullRequest.ReviewThreads
		all = append(all, page.Nodes...)
		if !page.PageInfo.HasNextPage {
			break
		}
		cursor = page.PageInfo.EndCursor
	}
	return all, nil
}
