package change_report

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type ChangeType string

var (
	ADDED   ChangeType = "ADDED"
	REMOVED ChangeType = "REMOVED"
	CONTEXT ChangeType = "CONTEXT"
)

type FileChange struct {
	Filename   string
	AddedLines map[int]string
}

type ChangeReport map[string]*FileChange

func GenerateChangeReport(baseRef string) (ChangeReport, error) {
	out, err := gitExec("diff", baseRef)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get git diff")
	}
	report, err := parseDiff(string(out))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse git diff")
	}

	return *report, nil
}

func parseDiff(diffString string) (*ChangeReport, error) {
	diff := make(ChangeReport)
	var file *FileChange
	var lineCount int
	var inHunk bool
	newFilePrefix := "+++ b/"
	isFileDeleted := false

	lines := strings.Split(diffString, "\n")
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff "):
			inHunk = false
			isFileDeleted = false
			file = &FileChange{}
		case isFileDeleted:
			continue
		case line == "+++ /dev/null":
			isFileDeleted = true
		case strings.HasPrefix(line, newFilePrefix):
			file.Filename = strings.TrimPrefix(line, newFilePrefix)
			file.AddedLines = make(map[int]string)
			diff[file.Filename] = file
		case strings.HasPrefix(line, "@@ "):
			inHunk = true

			re := regexp.MustCompile(`@@ \-(\d+),?(\d+)? \+(\d+),?(\d+)? @@`)
			m := re.FindStringSubmatch(line)
			if len(m) < 4 {
				return nil, errors.New("Error parsing line: " + line)
			}
			diffStartLine, err := strconv.Atoi(m[3])
			if err != nil {
				return nil, err
			}

			lineCount = diffStartLine
		case inHunk && isSourceLine(line):
			t, err := getChangeType(line)
			if err != nil {
				return nil, err
			}
			if *t != REMOVED {
				if *t == ADDED {
					file.AddedLines[lineCount] = line[1:]
				}
				lineCount++
			}
		}
	}

	return &diff, nil
}

func gitExec(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "failed run git cmd output: ")
	}

	return output, nil
}

func isSourceLine(line string) bool {
	if line == `\ No newline at end of file` {
		return false
	}
	if l := len(line); l == 0 || (l >= 3 && (line[:3] == "---" || line[:3] == "+++")) {
		return false
	}
	return true
}

func getChangeType(line string) (*ChangeType, error) {
	var t ChangeType
	switch line[:1] {
	case " ":
		t = CONTEXT
	case "+":
		t = ADDED
	case "-":
		t = REMOVED
	default:
		return nil, fmt.Errorf("failed to parse line mode for line: '%s'", line)
	}
	return &t, nil
}
