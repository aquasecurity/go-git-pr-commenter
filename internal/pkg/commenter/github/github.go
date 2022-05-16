package github

import (
	"fmt"
	"github.com/google/go-github/v44/github"
	"regexp"
	"strconv"
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
	patchRegex     = regexp.MustCompile(`^@@.*\d [\+\-](\d+),?(\d+)?.+?@@`)
	commitRefRegex = regexp.MustCompile(".+ref=(.+)")
)

func NewGithub(token, owner, repo string, prNumber int) (gh *Github, err error) {
	if len(token) == 0 {
		return gh, fmt.Errorf("failed GITHUB_TOKEN has not been set")
	}
	ghConnector, err := createConnector(token, owner, repo, prNumber)
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
	hunkStart, hunkEnd, err := parseHunkPositions(patch, *file.Filename)
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
		hunkStart:    hunkStart,
		hunkEnd:      hunkStart + (hunkEnd - 1),
		sha:          sha,
		likelyBinary: isBinary,
	}, nil
}
func parseHunkPositions(patch, filename string) (hunkStart int, hunkEnd int, err error) {
	if patch != "" {
		groups := patchRegex.FindAllStringSubmatch(patch, -1)
		if len(groups) < 1 {
			return 0, 0, fmt.Errorf("the patch details for [%s] could not be resolved", filename)
		}

		patchGroup := groups[0]
		endPos := 2
		if len(patchGroup) > 2 && patchGroup[2] == "" {
			endPos = 1
		}

		hunkStart, err = strconv.Atoi(patchGroup[1])
		if err != nil {
			hunkStart = -1
		}
		hunkEnd, err = strconv.Atoi(patchGroup[endPos])
		if err != nil {
			hunkEnd = -1
		}
	}
	return hunkStart, hunkEnd, nil
}

func (c *Github) checkCommentRelevant(filename string, line int) bool {

	for _, file := range c.files {
		if relevant := func(file *commitFileInfo) bool {
			if file.FileName == filename && !file.isResolvable() {
				if line >= file.hunkStart && line <= file.hunkEnd {
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
func (c *Github) getFileInfo(file string, line int) (*commitFileInfo, error) {

	for _, info := range c.files {
		if info.FileName == file && !info.isResolvable() {
			if line >= info.hunkStart && line <= info.hunkEnd {
				return info, nil
			}
		}
	}
	return nil, fmt.Errorf("file not found, shouldn't have got to here")
}
func buildComment(file, comment string, line int, info commitFileInfo) *github.PullRequestComment {

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
