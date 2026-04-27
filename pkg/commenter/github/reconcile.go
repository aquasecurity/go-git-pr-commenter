package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter"
	gh "github.com/google/go-github/v44/github"
)

type aquaThread struct {
	thread      *gqlReviewThread
	topComment  *gqlReviewComment
	fingerprint string
}

func (c *Github) ReconcileAquaComments(marker string, current []commenter.Finding) error {
	ctx := context.Background()
	threads, err := c.ghConnector.fetchReviewThreads(ctx, c.Token, c.GraphQLEndpoint)
	if err != nil {
		return fmt.Errorf("list review threads: %w", err)
	}

	aqua := selectAquaThreads(threads, marker)
	byFP, legacy := indexAquaThreads(aqua)

	handled := make(map[string]bool)
	legacyUsed := make(map[*aquaThread]bool)

	for _, f := range current {
		match := matchThread(f, byFP, legacy, legacyUsed)
		if match != nil {
			if f.Fingerprint != "" {
				handled[f.Fingerprint] = true
			}
			if match.thread.IsResolved {
				continue
			}
			if err := c.editComment(ctx, match.topComment.DatabaseID, f.Body); err != nil {
				return fmt.Errorf("edit comment %d: %w", match.topComment.DatabaseID, err)
			}
			continue
		}
		// No matching thread — fall through to the existing create path so we
		// inherit checkCommentRelevant, position calculation, and retries.
		_ = c.WriteMultiLineComment(f.Path, f.Body, f.StartLine, f.EndLine)
	}

	for fp, a := range byFP {
		if handled[fp] || a.thread.IsResolved {
			continue
		}
		if err := c.deleteAquaCommentsInThread(ctx, a.thread, marker); err != nil {
			return err
		}
	}
	for _, a := range legacy {
		if legacyUsed[a] || a.thread.IsResolved {
			continue
		}
		if err := c.deleteAquaCommentsInThread(ctx, a.thread, marker); err != nil {
			return err
		}
	}
	return nil
}

func selectAquaThreads(threads []gqlReviewThread, marker string) []*aquaThread {
	out := make([]*aquaThread, 0, len(threads))
	for i := range threads {
		t := &threads[i]
		var top *gqlReviewComment
		for j := range t.Comments.Nodes {
			if strings.Contains(t.Comments.Nodes[j].Body, marker) {
				top = &t.Comments.Nodes[j]
				break
			}
		}
		if top == nil {
			continue
		}
		out = append(out, &aquaThread{
			thread:      t,
			topComment:  top,
			fingerprint: ExtractFingerprint(top.Body),
		})
	}
	return out
}

func indexAquaThreads(aqua []*aquaThread) (byFP map[string]*aquaThread, legacy []*aquaThread) {
	byFP = make(map[string]*aquaThread)
	for _, a := range aqua {
		if a.fingerprint == "" {
			legacy = append(legacy, a)
			continue
		}
		// First-seen wins; later duplicates fall through to deletion below.
		if _, ok := byFP[a.fingerprint]; !ok {
			byFP[a.fingerprint] = a
		}
	}
	return byFP, legacy
}

func matchThread(f commenter.Finding, byFP map[string]*aquaThread, legacy []*aquaThread, used map[*aquaThread]bool) *aquaThread {
	if f.Fingerprint != "" {
		if a, ok := byFP[f.Fingerprint]; ok {
			return a
		}
	}
	for _, a := range legacy {
		if used[a] {
			continue
		}
		if a.thread.Path == f.Path && intPtrEq(a.thread.StartLine, f.StartLine) && intPtrEq(a.thread.Line, f.EndLine) {
			used[a] = true
			return a
		}
	}
	return nil
}

func intPtrEq(p *int, v int) bool {
	if p == nil {
		return v == 0
	}
	return *p == v
}

func (c *Github) editComment(ctx context.Context, id int64, body string) error {
	_, _, err := c.ghConnector.prs.EditComment(ctx, c.Owner, c.Repo, id, &gh.PullRequestComment{Body: &body})
	return err
}

// Only deletes comments authored by Aqua (i.e. carrying the marker) so that any
// developer replies in the thread are preserved.
func (c *Github) deleteAquaCommentsInThread(ctx context.Context, t *gqlReviewThread, marker string) error {
	for _, cm := range t.Comments.Nodes {
		if !strings.Contains(cm.Body, marker) {
			continue
		}
		if _, err := c.ghConnector.prs.DeleteComment(ctx, c.Owner, c.Repo, cm.DatabaseID); err != nil {
			return err
		}
	}
	return nil
}
