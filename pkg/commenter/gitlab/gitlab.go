package gitlab

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/utils"

	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter"
)

type DiscussionNote struct {
	DiscussionId string
	NoteId       int
}
type Discussion struct {
	Id    string `json:"id,omitempty"`
	Notes []Note `json:"notes,omitempty"`
}

type Note struct {
	Id   int    `json:"id,omitempty"`
	Body string `json:"body,omitempty"`
}

type Version struct {
	ID             int       `json:"id"`
	HeadCommitSha  string    `json:"head_commit_sha"`
	BaseCommitSha  string    `json:"base_commit_sha"`
	StartCommitSha string    `json:"start_commit_sha"`
	CreatedAt      time.Time `json:"created_at"`
	MergeRequestID int       `json:"merge_request_id"`
	State          string    `json:"state"`
	RealSize       string    `json:"real_size"`
}

type Gitlab struct {
	ApiURL   string
	Token    string
	Repo     string
	PrNumber string
}

func NewGitlab(token string) (b *Gitlab, err error) {
	return &Gitlab{
		ApiURL:   os.Getenv("CI_API_V4_URL"),
		Token:    token,
		Repo:     os.Getenv("CI_PROJECT_ID"),
		PrNumber: os.Getenv("CI_MERGE_REQUEST_IID"),
	}, nil
}

// WriteMultiLineComment writes a multiline review on a file in the gitlab PR
func (c *Gitlab) WriteMultiLineComment(file, comment string, startLine, _ int) error {
	// In gitlab we support one line only
	err := c.WriteLineComment(file, comment, startLine)
	if err != nil {
		return fmt.Errorf("failed write gitlab multi line comment: %w", err)
	}

	return nil
}

// WriteLineComment writes a single review line on a file of the gitlab PR
func (c *Gitlab) WriteLineComment(file, comment string, line int) error {
	if line == 0 {
		line = 1
	}

	version, err := c.getLatestVersion()
	if err != nil {
		return fmt.Errorf("failed get latest version: %w", err)
	}
	urlValues := url.Values{
		"position[position_type]": {"text"},
		"position[base_sha]":      {version.BaseCommitSha},
		"position[head_sha]":      {version.HeadCommitSha},
		"position[start_sha]":     {version.StartCommitSha},
		"position[new_path]":      {file},
		"position[new_line]":      {strconv.Itoa(line)},
		"body":                    {comment},
	}

	if line == commenter.FIRST_AVAILABLE_LINE {
		line = 1
		urlValues["position[new_line]"] = []string{strconv.Itoa(line)}
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/projects/%s/merge_requests/%s/discussions",
		c.ApiURL, c.Repo, c.PrNumber),
		strings.NewReader(urlValues.Encode()))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("PRIVATE-TOKEN", c.Token)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("failed to write comment to file: %s, trying again", file)
		urlValues["position[old_line]"] = []string{strconv.Itoa(line)}
		req, err := http.NewRequest("POST", fmt.Sprintf("%s/projects/%s/merge_requests/%s/discussions",
			c.ApiURL, c.Repo, c.PrNumber),
			strings.NewReader(urlValues.Encode()))
		if err != nil {
			return err
		}
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("PRIVATE-TOKEN", c.Token)
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusCreated {
			b, _ := ioutil.ReadAll(resp.Body)
			return fmt.Errorf("failed to write comment to file: %s, on line: %d, with gitlab error: %s", file, line, string(b))
		}

		fmt.Println("comment created successfully")
	}

	return nil
}

func (c *Gitlab) getLatestVersion() (v Version, err error) {
	var vData []Version

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/projects/%s/merge_requests/%s/versions",
		c.ApiURL, c.Repo, c.PrNumber), nil)
	if err != nil {
		return v, err
	}
	req.Header.Add("PRIVATE-TOKEN", c.Token)
	resp, err := client.Do(req)
	if err != nil {
		return v, err
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(resp.Body)
		return v, fmt.Errorf("failed get gitlab PR version: %s", string(b))
	}
	defer func() { _ = resp.Body.Close() }()
	err = json.NewDecoder(resp.Body).Decode(&vData)
	if err != nil {
		return v, fmt.Errorf("failed decoding gitlab version response with error: %w", err)
	}

	if len(vData) > 0 {
		v = vData[0]
	}
	return v, nil
}

func (c *Gitlab) RemovePreviousAquaComments(msg string) error {

	var idsToRemove []DiscussionNote
	idsToRemove, err := c.getIdsToRemove(idsToRemove, msg, "1")
	if err != nil {
		return err
	}

	for _, idToRemove := range idsToRemove {
		err = utils.DeleteComments(fmt.Sprintf("%s/projects/%s/merge_requests/%s/discussions/%s/notes/%s",
			c.ApiURL, c.Repo, c.PrNumber, idToRemove.DiscussionId, strconv.Itoa(idToRemove.NoteId)), map[string]string{"PRIVATE-TOKEN": c.Token})
		if err != nil {
			return err
		}
	}

	return nil

}

func (c *Gitlab) getIdsToRemove(idsToRemove []DiscussionNote, msg, page string) ([]DiscussionNote, error) {
	resp, err := utils.GetComments(
		fmt.Sprintf("%s/projects/%s/merge_requests/%s/discussions?page=%s",
			c.ApiURL,
			c.Repo,
			c.PrNumber,
			page),
		map[string]string{"PRIVATE-TOKEN": c.Token})
	if err != nil {
		return nil, fmt.Errorf("failed getting comments with error: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var discussionsResponse []Discussion
	err = json.Unmarshal(body, &discussionsResponse)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshal response body with error: %w", err)
	}

	for _, discussion := range discussionsResponse {
		for _, note := range discussion.Notes {
			if strings.Contains(note.Body, msg) {
				idsToRemove = append(idsToRemove, DiscussionNote{
					DiscussionId: discussion.Id,
					NoteId:       note.Id,
				})
			}
		}
	}

	if resp.Header.Get("x-next-page") == "" {
		return idsToRemove, nil
	}
	return c.getIdsToRemove(idsToRemove, msg, resp.Header.Get("x-next-page"))

}
