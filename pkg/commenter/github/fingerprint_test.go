package github

import "testing"

func TestEmbedFingerprint(t *testing.T) {
	tests := []struct {
		name string
		body string
		fp   string
		want string
	}{
		{
			name: "appends sentinel on its own line",
			body: "hello",
			fp:   "deadbeef",
			want: "hello\n<!-- aqua-fingerprint: deadbeef -->",
		},
		{
			name: "reuses existing trailing newline",
			body: "hello\n",
			fp:   "deadbeef",
			want: "hello\n<!-- aqua-fingerprint: deadbeef -->",
		},
		{
			name: "no-op when fp empty",
			body: "hello",
			fp:   "",
			want: "hello",
		},
		{
			name: "no-op when sentinel already present",
			body: "hello\n<!-- aqua-fingerprint: cafebabe -->",
			fp:   "deadbeef",
			want: "hello\n<!-- aqua-fingerprint: cafebabe -->",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := EmbedFingerprint(tc.body, tc.fp)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractFingerprint(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		{"<!-- aqua-fingerprint: deadbeef -->", "deadbeef"},
		{"prefix\n<!--aqua-fingerprint:CAFEBABE-->\nsuffix", "cafebabe"},
		{"prefix\n<!--   aqua-fingerprint:   abc123   -->\nsuffix", "abc123"},
		{"no sentinel here", ""},
		{"<!-- aqua-fingerprint: not_hex_chars -->", ""},
	}
	for _, tc := range tests {
		got := ExtractFingerprint(tc.body)
		if got != tc.want {
			t.Errorf("ExtractFingerprint(%q) = %q, want %q", tc.body, got, tc.want)
		}
	}
}
