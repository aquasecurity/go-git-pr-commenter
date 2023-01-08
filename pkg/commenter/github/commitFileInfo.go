package github

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
)

type chunkLines struct {
	Start int
	End   int
}

type commitFileInfo struct {
	FileName     string
	ChunkLines   []chunkLines
	sha          string
	likelyBinary bool
}

func (cl *chunkLines) Contains(line int) bool {
	return line >= cl.Start && line <= cl.End
}

func getCommitFileInfo(ghConnector *connector) ([]*commitFileInfo, error) {

	prFiles, err := ghConnector.getFilesForPr()
	if err != nil {
		return nil, err
	}

	var (
		errs            []string
		commitFileInfos []*commitFileInfo
	)

	for _, file := range prFiles {
		info, err := getCommitInfo(file)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		commitFileInfos = append(commitFileInfos, info)
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("there were errors processing the PR files.\n%s", strings.Join(errs, "\n"))
	}
	return commitFileInfos, nil
}

func (cfi commitFileInfo) calculatePosition(line int) *int {
	ch, _ := lo.Find(cfi.ChunkLines, func(lines chunkLines) bool {
		return lines.Contains(line)
	})

	position := line - ch.Start
	return &position
}

func (cfi commitFileInfo) isBinary() bool {
	return cfi.likelyBinary
}

func (cfi commitFileInfo) isResolvable() bool {
	return cfi.isBinary()
}
