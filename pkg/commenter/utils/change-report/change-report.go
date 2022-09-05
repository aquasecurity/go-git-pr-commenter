package change_report

import (
	"os/exec"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/sourcegraph/go-diff/diff"
)

type ChangeType string

var (
	ADDED   ChangeType = "ADDED"
	REMOVED ChangeType = "REMOVED"
	CONTEXT ChangeType = "CONTEXT"
)

type FileChange struct {
	StartLine  int
	EndLine    int
	ChangeType ChangeType
}

type FileChanges []FileChange

type ChangeReport map[string]FileChanges

func GenerateChangeReport(baseRef string) (ChangeReport, error) {
	out, err := GitExec("diff", baseRef)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get git diff")
	}

	changeReport := make(ChangeReport)
	multiDiff, _ := diff.ParseMultiFileDiff(out)

	for _, fileDiff := range multiDiff {
		filename := fileDiff.NewName
		fileChanges := lo.Map(fileDiff.Hunks, toFileChange)
		changeReport[filename] = fileChanges
	}

	return changeReport, nil
}

func toFileChange(hunk *diff.Hunk, _ int) FileChange {
	linesChange := hunk.NewLines - hunk.OrigLines
	changeType := lo.Ternary(linesChange > 0, ADDED, lo.Ternary(linesChange < 0, REMOVED, CONTEXT))

	return FileChange{
		StartLine:  int(hunk.NewStartLine),
		EndLine:    int(hunk.NewStartLine + hunk.NewLines - 1),
		ChangeType: changeType,
	}
}

func GitExec(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "failed run git cmd output: ")
	}

	return output, nil
}
