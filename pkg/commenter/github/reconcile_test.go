package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter"
	gh "github.com/google/go-github/v44/github"
)

const testMarker = "[This comment was created by Aqua Pipeline]"

type apiCounts struct {
	graphql, edit, delete, create int32
}

type gqlThreadFixture struct {
	resolved    bool
	path        string
	line        int
	fingerprint string
	body        string
	commentID   int64
}

// renderGraphQLResponse marshals fixtures into the same shape the real GitHub
// GraphQL API would return.
func renderGraphQLResponse(threads []gqlThreadFixture) string {
	type cmt struct {
		DatabaseID int64  `json:"databaseId"`
		Body       string `json:"body"`
		Path       string `json:"path"`
		Line       *int   `json:"line"`
		StartLine  *int   `json:"startLine"`
	}
	type thd struct {
		ID         string `json:"id"`
		IsResolved bool   `json:"isResolved"`
		IsOutdated bool   `json:"isOutdated"`
		Path       string `json:"path"`
		Line       *int   `json:"line"`
		StartLine  *int   `json:"startLine"`
		Comments   struct {
			Nodes []cmt `json:"nodes"`
		} `json:"comments"`
	}
	out := struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					ReviewThreads struct {
						PageInfo struct {
							HasNextPage bool   `json:"hasNextPage"`
							EndCursor   string `json:"endCursor"`
						} `json:"pageInfo"`
						Nodes []thd `json:"nodes"`
					} `json:"reviewThreads"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}{}
	for _, t := range threads {
		body := t.body
		if t.fingerprint != "" && !strings.Contains(body, "aqua-fingerprint") {
			body = EmbedFingerprint(body, t.fingerprint)
		}
		line := t.line
		nodes := []cmt{{DatabaseID: t.commentID, Body: body, Path: t.path, Line: &line, StartLine: &line}}
		out.Data.Repository.PullRequest.ReviewThreads.Nodes = append(
			out.Data.Repository.PullRequest.ReviewThreads.Nodes,
			thd{
				ID:         fmt.Sprintf("PRT_%d", t.commentID),
				IsResolved: t.resolved,
				Path:       t.path,
				Line:       &line,
				StartLine:  &line,
				Comments: struct {
					Nodes []cmt `json:"nodes"`
				}{Nodes: nodes},
			},
		)
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func newTestGithub(t *testing.T, threads []gqlThreadFixture, commitFiles []*commitFileInfo) (*Github, *apiCounts, func()) {
	t.Helper()
	counts := &apiCounts{}
	mux := http.NewServeMux()

	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&counts.graphql, 1)
		w.Header().Set("Content-Type", "application/json")
		body := renderGraphQLResponse(threads)
		_, _ = w.Write([]byte(body))
	})
	// PATCH/DELETE on a single review comment.
	mux.HandleFunc("/repos/owner/repo/pulls/comments/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPatch:
			atomic.AddInt32(&counts.edit, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":1}`))
		case http.MethodDelete:
			atomic.AddInt32(&counts.delete, 1)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s on %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/repos/owner/repo/pulls/42/comments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			atomic.AddInt32(&counts.create, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":999}`))
			return
		}
		t.Errorf("unexpected method %s on %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	ts := httptest.NewServer(mux)

	client := gh.NewClient(nil)
	base, _ := url.Parse(ts.URL + "/")
	client.BaseURL = base

	c := &Github{
		Token:           "x",
		Owner:           "owner",
		Repo:            "repo",
		PrNumber:        42,
		GraphQLEndpoint: ts.URL + "/graphql",
		files:           commitFiles,
		ghConnector: &connector{
			prs:      client.PullRequests,
			owner:    "owner",
			repo:     "repo",
			prNumber: 42,
		},
	}
	return c, counts, ts.Close
}

func filesCovering(path string, start, end int) []*commitFileInfo {
	return []*commitFileInfo{{
		FileName:   path,
		sha:        "abc",
		ChunkLines: []chunkLines{{Start: start, End: end}},
	}}
}

func aquaBody(extra string) string {
	return extra + "\n" + testMarker
}

func TestReconcile_ResolvedThreadKept_NoApiWrites(t *testing.T) {
	c, counts, done := newTestGithub(t,
		[]gqlThreadFixture{{
			resolved:    true,
			path:        "a.go",
			line:        10,
			commentID:   100,
			fingerprint: "deadbeef",
			body:        aquaBody("old finding text"),
		}},
		filesCovering("a.go", 1, 100),
	)
	defer done()

	err := c.ReconcileAquaComments(testMarker, []commenter.Finding{{
		Path: "a.go", StartLine: 10, EndLine: 10,
		Body:        EmbedFingerprint(aquaBody("refreshed finding text"), "deadbeef"),
		Fingerprint: "deadbeef",
	}})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if counts.edit != 0 || counts.delete != 0 || counts.create != 0 {
		t.Fatalf("expected zero writes, got edit=%d delete=%d create=%d", counts.edit, counts.delete, counts.create)
	}
}

func TestReconcile_UnresolvedSameFinding_EditsInPlace(t *testing.T) {
	c, counts, done := newTestGithub(t,
		[]gqlThreadFixture{{
			resolved:    false,
			path:        "a.go",
			line:        10,
			commentID:   100,
			fingerprint: "deadbeef",
			body:        aquaBody("old finding text"),
		}},
		filesCovering("a.go", 1, 100),
	)
	defer done()

	err := c.ReconcileAquaComments(testMarker, []commenter.Finding{{
		Path: "a.go", StartLine: 10, EndLine: 10,
		Body:        EmbedFingerprint(aquaBody("refreshed finding text"), "deadbeef"),
		Fingerprint: "deadbeef",
	}})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if counts.edit != 1 || counts.delete != 0 || counts.create != 0 {
		t.Fatalf("expected edit=1, got edit=%d delete=%d create=%d", counts.edit, counts.delete, counts.create)
	}
}

func TestReconcile_UnresolvedStaleFinding_Deletes(t *testing.T) {
	c, counts, done := newTestGithub(t,
		[]gqlThreadFixture{{
			resolved:    false,
			path:        "a.go",
			line:        10,
			commentID:   100,
			fingerprint: "cafebabe",
			body:        aquaBody("finding gone in latest scan"),
		}},
		filesCovering("a.go", 1, 100),
	)
	defer done()

	if err := c.ReconcileAquaComments(testMarker, nil); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if counts.delete != 1 || counts.edit != 0 || counts.create != 0 {
		t.Fatalf("expected delete=1, got edit=%d delete=%d create=%d", counts.edit, counts.delete, counts.create)
	}
}

func TestReconcile_ResolvedStaleFinding_Preserved(t *testing.T) {
	c, counts, done := newTestGithub(t,
		[]gqlThreadFixture{{
			resolved:    true,
			path:        "a.go",
			line:        10,
			commentID:   100,
			fingerprint: "cafebabe",
			body:        aquaBody("accepted-risk finding gone too"),
		}},
		filesCovering("a.go", 1, 100),
	)
	defer done()

	if err := c.ReconcileAquaComments(testMarker, nil); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if counts.edit != 0 || counts.delete != 0 || counts.create != 0 {
		t.Fatalf("expected no writes, got edit=%d delete=%d create=%d", counts.edit, counts.delete, counts.create)
	}
}

func TestReconcile_NewFinding_CreatesComment(t *testing.T) {
	c, counts, done := newTestGithub(t, nil, filesCovering("a.go", 1, 100))
	defer done()

	err := c.ReconcileAquaComments(testMarker, []commenter.Finding{{
		Path: "a.go", StartLine: 10, EndLine: 20,
		Body:        EmbedFingerprint(aquaBody("brand new"), "feedface"),
		Fingerprint: "feedface",
	}})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if counts.create != 1 || counts.edit != 0 || counts.delete != 0 {
		t.Fatalf("expected create=1, got edit=%d delete=%d create=%d", counts.edit, counts.delete, counts.create)
	}
}

func TestReconcile_LegacyThread_MatchedByPathAndLine(t *testing.T) {
	c, counts, done := newTestGithub(t,
		[]gqlThreadFixture{{
			resolved:    false,
			path:        "a.go",
			line:        10,
			commentID:   100,
			fingerprint: "", // legacy: no fingerprint embedded
			body:        aquaBody("old aqua comment from a previous scanner version"),
		}},
		filesCovering("a.go", 1, 100),
	)
	defer done()

	err := c.ReconcileAquaComments(testMarker, []commenter.Finding{{
		Path: "a.go", StartLine: 10, EndLine: 10,
		Body:        EmbedFingerprint(aquaBody("now with fingerprint"), "deadbeef"),
		Fingerprint: "deadbeef",
	}})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if counts.edit != 1 || counts.create != 0 || counts.delete != 0 {
		t.Fatalf("expected edit=1, got edit=%d delete=%d create=%d", counts.edit, counts.delete, counts.create)
	}
}

func TestReconcile_NonAquaThread_Ignored(t *testing.T) {
	c, counts, done := newTestGithub(t,
		[]gqlThreadFixture{{
			resolved:  false,
			path:      "a.go",
			line:      10,
			commentID: 100,
			body:      "human reviewer comment with no aqua marker",
		}},
		filesCovering("a.go", 1, 100),
	)
	defer done()

	if err := c.ReconcileAquaComments(testMarker, nil); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if counts.delete != 0 || counts.edit != 0 || counts.create != 0 {
		t.Fatalf("non-aqua thread must not be touched, got edit=%d delete=%d create=%d", counts.edit, counts.delete, counts.create)
	}
}
