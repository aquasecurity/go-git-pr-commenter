package gitlab

import "testing"

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
			name: "line_code error on regular source file",
			file: "src/main.go",
			gitlabError: `{"message":"400 Bad request - Note {:line_code=>[\"can't be blank\", \"must be a valid line code\"]}"}`,
			want: true,
		},
		{
			name: "unrelated error on regular source file",
			file: "src/main.go",
			gitlabError: `{"message":"401 Unauthorized"}`,
			want: false,
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
