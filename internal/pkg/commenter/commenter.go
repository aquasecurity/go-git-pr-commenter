package commenter

type GitCommenter struct {
	Token    string
	Owner    string
	Repo     string
	PrNumber int
}

type Repository interface {
	// WriteMultiLineComment writes a multiline review on a file in the git PR
	WriteMultiLineComment(file, comment string, startLine, endLine string) error
	// WriteLineComment writes a single review line on a file of the git PR
	WriteLineComment(file, comment, line string) error
}
