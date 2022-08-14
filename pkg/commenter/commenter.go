package commenter

type Repository interface {
	// WriteMultiLineComment writes a multiline review on a file in the git PR
	WriteMultiLineComment(file, comment string, startLine, endLine int) error
	// WriteLineComment writes a single review line on a file of the git PR
	WriteLineComment(file, comment string, line int) error
}

var FIRST_AVAILABLE_LINE = -1
