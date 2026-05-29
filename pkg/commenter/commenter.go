package commenter

type Repository interface {
	// WriteMultiLineComment writes a multiline review on a file in the git PR
	WriteMultiLineComment(file, comment string, startLine, endLine int) error
	// WriteLineComment writes a single review line on a file of the git PR
	WriteLineComment(file, comment string, line int) error
	// RemovePreviousAquaComments Removing the comments from previous PRs
	RemovePreviousAquaComments(msg string) error
}

var FIRST_AVAILABLE_LINE = -1

// Finding is one logical scanner result. Body must already contain both the
// Aqua marker and the fingerprint sentinel (see EmbedFingerprint), so that
// reconciliation can identify and match it across runs.
type Finding struct {
	Path        string
	StartLine   int
	EndLine     int
	Body        string
	Fingerprint string
}

// Reconciler is an optional capability detected via type assertion; providers
// that don't implement it fall back to the legacy delete-all + repost path.
type Reconciler interface {
	ReconcileAquaComments(marker string, current []Finding) error
}
