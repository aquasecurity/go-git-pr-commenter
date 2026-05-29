package github

import (
	"regexp"
	"strings"
)

const (
	fingerprintPrefix = "<!-- aqua-fingerprint:"
	fingerprintSuffix = "-->"
)

// HTML comment so it stays invisible in every Markdown renderer GitHub uses.
var fingerprintRe = regexp.MustCompile(`<!--\s*aqua-fingerprint:\s*([0-9a-fA-F]+)\s*-->`)

func EmbedFingerprint(body, fp string) string {
	if fp == "" || fingerprintRe.MatchString(body) {
		return body
	}
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return body + fingerprintPrefix + " " + fp + " " + fingerprintSuffix
}

func ExtractFingerprint(body string) string {
	m := fingerprintRe.FindStringSubmatch(body)
	if len(m) < 2 {
		return ""
	}
	return strings.ToLower(m[1])
}
