package gitlab

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

func TestShouldUseGeneralPrComment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		file        string
		gitlabError string
		want        bool
	}{
		{
			name: "nuget lock file",
			file: "ELMS.Api.Tests/packages.lock.json",
			want: true,
		},
		{
			name: "npm lock file via package.json basename match",
			file: "e2e/package-lock.json",
			want: true,
		},
		{
			name: "yarn lock file",
			file: "frontend/yarn.lock",
			want: true,
		},
		{
			name: "line_code error on regular source file stays inline failure",
			file: "src/main.go",
			gitlabError: `{"message":"400 Bad request - Note {:line_code=>[\"can't be blank\", \"must be a valid line code\"]}"}`,
			want: false,
		},
		{
			name: "line_code error on lockfile still falls back",
			file: "ELMS.Api.Tests/packages.lock.json",
			gitlabError: `{"message":"400 Bad request - Note {:line_code=>[\"can't be blank\", \"must be a valid line code\"]}"}`,
			want: true,
		},
		{
			name: "unrelated error on regular source file",
			file: "src/main.go",
			gitlabError: `{"message":"401 Unauthorized"}`,
			want: false,
		},
		{
			name:        "elevated 400 from versions API",
			file:        "src/main.go",
			gitlabError: "failed get gitlab PR version: The service is temporarily unavailable due to elevated 400 error rates. Please retry after a short wait.",
			want:        true,
		},
		{
			name:        "rate limit response",
			file:        "src/main.go",
			gitlabError: "429 Too Many Requests",
			want:        true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := shouldUseGeneralPrComment(tt.file, tt.gitlabError); got != tt.want {
				t.Fatalf("shouldUseGeneralPrComment(%q, %q) = %v, want %v", tt.file, tt.gitlabError, got, tt.want)
			}
		})
	}
}

func TestWriteLineCommentEmptySkipsGitLabAPI(t *testing.T) {
	t.Parallel()

	// A closed port fails immediately if any request is attempted.
	c := &Gitlab{
		ApiURL:   "http://127.0.0.1:1",
		Token:    "token",
		Repo:     "1",
		PrNumber: "1",
	}

	comments := []string{"", " ", "\t", "\n", " \n\t "}
	files := []string{"src/main.go", "ELMS.Api.Tests/packages.lock.json"}
	for _, file := range files {
		for _, comment := range comments {
			if err := c.WriteLineComment(file, comment, 10); err != nil {
				t.Fatalf("WriteLineComment(%q, %q) = %v, want nil without a GitLab call", file, comment, err)
			}
			if err := c.WriteMultiLineComment(file, comment, 10, 12); err != nil {
				t.Fatalf("WriteMultiLineComment(%q, %q) = %v, want nil without a GitLab call", file, comment, err)
			}
		}
	}
}

func TestWriteLineCommentFallsBackToMRNote(t *testing.T) {
	t.Parallel()

	const comment = "CVE-1 found"
	const file = "src/main.go"

	t.Run("inline failure then note created", func(t *testing.T) {
		t.Parallel()

		calls := serveGitLab(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions"):
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, `[{"id":1,"head_commit_sha":"head","base_commit_sha":"base","start_commit_sha":"start"}]`)
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/discussions"):
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, "The service is temporarily unavailable due to elevated 400 error rates. Please retry after a short wait.")
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/notes"):
				w.WriteHeader(http.StatusCreated)
			default:
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		})

		c := &Gitlab{ApiURL: calls.server.URL, Token: "token", Repo: "42", PrNumber: "7"}
		if err := c.WriteLineComment(file, comment, 12); err != nil {
			t.Fatalf("WriteLineComment() = %v, want nil", err)
		}

		calls.expect(t, []string{
			"GET /projects/42/merge_requests/7/versions",
			"POST /projects/42/merge_requests/7/discussions",
			"POST /projects/42/merge_requests/7/discussions",
			"POST /projects/42/merge_requests/7/notes",
		})

		retry := calls.bodies[2]
		vals, err := url.ParseQuery(retry)
		if err != nil {
			t.Fatalf("parse retry body: %v", err)
		}
		if vals.Get("position[old_line]") != "12" || vals.Get("position[new_line]") != "12" {
			t.Fatalf("retry position = old %q new %q", vals.Get("position[old_line]"), vals.Get("position[new_line]"))
		}

		noteVals, err := url.ParseQuery(calls.bodies[3])
		if err != nil {
			t.Fatalf("parse note body: %v", err)
		}
		note := noteVals.Get("body")
		if !strings.Contains(note, file) || !strings.Contains(note, comment) {
			t.Fatalf("note body = %q", note)
		}
		if !c.versionsUnavailable {
			t.Fatal("elevated 400 response should skip later versions calls")
		}
	})

	t.Run("note creation failure is returned", func(t *testing.T) {
		t.Parallel()

		calls := serveGitLab(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions"):
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, `[{"id":1,"head_commit_sha":"head","base_commit_sha":"base","start_commit_sha":"start"}]`)
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/discussions"):
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, "429 Too Many Requests")
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/notes"):
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, "notes unavailable")
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		})

		c := &Gitlab{ApiURL: calls.server.URL, Token: "token", Repo: "42", PrNumber: "7"}
		err := c.WriteLineComment(file, comment, 12)
		if err == nil || !strings.Contains(err.Error(), "failed to write gitlab MR note") || !strings.Contains(err.Error(), "notes unavailable") {
			t.Fatalf("WriteLineComment() = %v, want the /notes response", err)
		}
		if strings.Contains(err.Error(), "429 Too Many Requests") {
			t.Fatalf("WriteLineComment() = %v, reported the discussion response", err)
		}
		calls.expectSuffix(t, "/notes")
	})
}

type gitLabCalls struct {
	server *httptest.Server
	mu     sync.Mutex
	paths  []string
	bodies []string
}

func serveGitLab(t *testing.T, handle func(http.ResponseWriter, *http.Request)) *gitLabCalls {
	t.Helper()

	recorded := &gitLabCalls{}
	recorded.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != "token" {
			t.Errorf("PRIVATE-TOKEN = %q", r.Header.Get("PRIVATE-TOKEN"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		recorded.mu.Lock()
		recorded.paths = append(recorded.paths, r.Method+" "+r.URL.Path)
		recorded.bodies = append(recorded.bodies, string(body))
		recorded.mu.Unlock()
		handle(w, r)
	}))
	t.Cleanup(recorded.server.Close)
	return recorded
}

func (g *gitLabCalls) expect(t *testing.T, want []string) {
	t.Helper()
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.paths) != len(want) {
		t.Fatalf("requests = %v, want %v", g.paths, want)
	}
	for i := range want {
		if g.paths[i] != want[i] {
			t.Fatalf("requests = %v, want %v", g.paths, want)
		}
	}
}

func (g *gitLabCalls) expectSuffix(t *testing.T, suffix string) {
	t.Helper()
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, path := range g.paths {
		if strings.HasSuffix(path, suffix) {
			return
		}
	}
	t.Fatalf("requests = %v, want one ending in %s", g.paths, suffix)
}

func TestIsGitLabUnavailable(t *testing.T) {
	t.Parallel()

	if !isGitLabUnavailable("The service is temporarily unavailable due to elevated 400 error rates. Please retry after a short wait.") {
		t.Fatal("expected elevated 400 response to be treated as unavailable")
	}
	if isGitLabUnavailable(`{"message":"400 Bad request - Note {:line_code=>[\"can't be blank\"]}"}`) {
		t.Fatal("line_code validation errors are not a versions API outage")
	}
}
