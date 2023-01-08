package github

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter"
	"github.com/google/go-github/v44/github"
	"github.com/samber/lo"
)

type Github struct {
	ghConnector      *connector
	existingComments []*existingComment
	files            []*commitFileInfo
	Token            string
	Owner            string
	Repo             string
	PrNumber         int
}

var (
	patchRegex     = regexp.MustCompile(`@@.*\d [\+\-](\d+),?(\d+)?.+?@@`)
	commitRefRegex = regexp.MustCompile(".+ref=(.+)")
)

func NewGithub(token, owner, repo string, prNumber int) (gh *Github, err error) {
	if len(token) == 0 {
		return gh, fmt.Errorf("failed GITHUB_TOKEN has not been set")
	}
	ghConnector, err := createConnector("", token, owner, repo, prNumber, false)
	if err != nil {
		return gh, fmt.Errorf("failed create github connector: %w", err)
	}
	commitFileInfos, existingComments, err := loadPr(ghConnector)
	if err != nil {
		return nil, fmt.Errorf("failed load pr: %w", err)
	}
	return &Github{
		Token:            token,
		Owner:            owner,
		PrNumber:         prNumber,
		Repo:             repo,
		ghConnector:      ghConnector,
		files:            commitFileInfos,
		existingComments: existingComments,
	}, nil
}

func NewGithubServer(apiUrl, token, owner, repo string, prNumber int) (gh *Github, err error) {
	if len(token) == 0 {
		return gh, fmt.Errorf("failed GITHUB_TOKEN has not been set, for github Enterprise")
	}
	ghConnector, err := createConnector(apiUrl, token, owner, repo, prNumber, true)
	if err != nil {
		return gh, fmt.Errorf("failed create github connector, for github Enterprise: %w", err)
	}
	commitFileInfos, existingComments, err := loadPr(ghConnector)
	if err != nil {
		return nil, fmt.Errorf("failed load pr, for github Enterprise: %w", err)
	}
	return &Github{
		Token:            token,
		Owner:            owner,
		PrNumber:         prNumber,
		Repo:             repo,
		ghConnector:      ghConnector,
		files:            commitFileInfos,
		existingComments: existingComments,
	}, nil
}
func loadPr(ghConnector *connector) ([]*commitFileInfo, []*existingComment, error) {

	commitFileInfos, err := getCommitFileInfo(ghConnector)
	if err != nil {
		return nil, nil, err
	}

	existingComments, err := ghConnector.getExistingComments()
	if err != nil {
		return nil, nil, err
	}
	return commitFileInfos, existingComments, nil
}

func getCommitInfo(file *github.CommitFile) (cfi *commitFileInfo, err error) {
	var isBinary bool
	patch := file.GetPatch()
	lines, err := parseChunkPositions(patch, *file.Filename)
	if err != nil {
		return nil, err
	}

	shaGroups := commitRefRegex.FindAllStringSubmatch(file.GetContentsURL(), -1)
	if len(shaGroups) < 1 {
		return nil, fmt.Errorf("the sha details for [%s] could not be resolved", *file.Filename)
	}
	sha := shaGroups[0][1]

	return &commitFileInfo{
		FileName:     *file.Filename,
		ChunkLines:   lines,
		sha:          sha,
		likelyBinary: isBinary,
	}, nil
}
func parseChunkPositions(patch, filename string) (lines []chunkLines, err error) {
	if patch != "" {
		groups := patchRegex.FindAllStringSubmatch(patch, -1)
		if len(groups) < 1 {
			return nil, fmt.Errorf("the patch details for [%s] could not be resolved", filename)
		}

		for _, patchGroup := range groups {
			endPos := 2
			if len(patchGroup) > 2 && patchGroup[2] == "" {
				endPos = 1
			}

			chunkStart, err := strconv.Atoi(patchGroup[1])
			if err != nil {
				chunkStart = -1
			}
			chunkEnd, err := strconv.Atoi(patchGroup[endPos])
			if err != nil {
				chunkEnd = -1
			}

			lines = append(lines, chunkLines{chunkStart, chunkStart + (chunkEnd - 1)})
		}
	}
	return lines, nil
}

func (c *Github) checkCommentRelevant(filename string, line int) bool {

	for _, file := range c.files {
		if relevant := func(file *commitFileInfo) bool {
			if file.FileName == filename && !file.isResolvable() {
				if (line == commenter.FIRST_AVAILABLE_LINE) || (checkIfLineInChunk(line, file)) {
					return true
				}
			}
			return false
		}(file); relevant {
			return true
		}
	}
	return false
}

func checkIfLineInChunk(line int, file *commitFileInfo) bool {
	if file.FileName == "go.mod" && len(file.ChunkLines) > 0 {
		return true
	}

	for _, lines := range file.ChunkLines {
		if lines.Contains(line) {
			return true
		}
	}
	return false
}

func (c *Github) getFileInfo(file string, line int) (*commitFileInfo, error) {

	for _, info := range c.files {
		if info.FileName == file && !info.isResolvable() {
			if (line == commenter.FIRST_AVAILABLE_LINE) || (checkIfLineInChunk(line, info)) {
				return info, nil
			}
		}
	}
	return nil, fmt.Errorf("file not found, shouldn't have got to here")
}

func getFirstChunkLine(file commitFileInfo) int {
	lines := lo.MinBy(file.ChunkLines, func(lines chunkLines, minLines chunkLines) bool {
		return lines.Start < minLines.Start

	})
	return lines.Start
}

func buildComment(file, comment string, line int, info commitFileInfo) *github.PullRequestComment {
	if line == commenter.FIRST_AVAILABLE_LINE {
		line = getFirstChunkLine(info)
	}

	return &github.PullRequestComment{
		Line:     &line,
		Path:     &file,
		CommitID: &info.sha,
		Body:     &comment,
		Position: info.calculatePosition(line),
	}
}

func (c *Github) writeCommentIfRequired(prComment *github.PullRequestComment) error {
	var commentId *int64
	for _, existing := range c.existingComments {
		commentId = func(ec *existingComment) *int64 {
			if *ec.filename == *prComment.Path && *ec.comment == *prComment.Body {
				return ec.commentId
			}
			return nil
		}(existing)
		if commentId != nil {
			break
		}
	}

	if err := c.ghConnector.writeReviewComment(prComment, commentId); err != nil {
		return fmt.Errorf("write review comment: %w", err)
	}
	return nil
}

// WriteMultiLineComment writes a multiline review on a file in the github PR
func (c *Github) WriteMultiLineComment(file, comment string, startLine, endLine int) error {
	if startLine == 0 {
		startLine = 1
	}
	if endLine == 0 {
		endLine = 1
	}

	if !c.checkCommentRelevant(file, startLine) || !c.checkCommentRelevant(file, endLine) {
		return newCommentNotValidError(file, startLine)
	}
	if startLine == endLine {
		return c.WriteLineComment(file, comment, endLine)
	}

	info, err := c.getFileInfo(file, endLine)
	if err != nil {
		return err
	}
	prComment := buildComment(file, comment, endLine, *info)
	prComment.StartLine = &startLine
	return c.writeCommentIfRequired(prComment)
}

// WriteLineComment writes a single review line on a file of the github PR
func (c *Github) WriteLineComment(file, comment string, line int) error {
	if !c.checkCommentRelevant(file, line) {
		return newCommentNotValidError(file, line)
	}
	info, err := c.getFileInfo(file, line)
	if err != nil {
		return err
	}
	prComment := buildComment(file, comment, line, *info)

	return c.writeCommentIfRequired(prComment)
}

func (c *Github) RemovePreviousAquaComments(_ string) error {
	return nil
}
