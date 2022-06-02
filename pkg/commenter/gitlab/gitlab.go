package gitlab

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

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
	if len(token) == 0 {
		return b, fmt.Errorf("failed GITLAB_TOKEN has not been set")
	}
	return &Gitlab{
		ApiURL:   os.Getenv("CI_API_V4_URL"),
		Token:    token,
		Repo:     os.Getenv("CI_PROJECT_ID"),
		PrNumber: os.Getenv("CI_MERGE_REQUEST_IID"),
	}, nil
}

// WriteMultiLineComment writes a multiline review on a file in the github PR
func (c *Gitlab) WriteMultiLineComment(file, comment string, startLine, endLine int) error {
	// In gitlab we support one line only
	err := c.WriteLineComment(file, comment, startLine)
	if err != nil {
		return fmt.Errorf("failed write gitlab multi line comment: %w", err)
	}

	return nil
}

// WriteLineComment writes a single review line on a file of the gitlab PR
func (c *Gitlab) WriteLineComment(file, comment string, line int) error {

	version, err := c.getLatestVersion()
	if err != nil {
		return fmt.Errorf("failed get latest version: %w", err)
	}
	if err != nil {
		return err
	}
	aa := url.Values{
		"position[position_type]": {"text"},
		"position[base_sha]":      {version.BaseCommitSha},
		"position[head_sha]":      {version.HeadCommitSha},
		"position[start_sha]":     {version.StartCommitSha},
		"position[new_path]":      {file},
		"position[new_line]":      {strconv.Itoa(line)},
		"body":                    {comment},
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/projects/%s/merge_requests/%s/discussions",
		c.ApiURL, c.Repo, c.PrNumber),
		strings.NewReader(aa.Encode()))
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
		return fmt.Errorf("failed write gitlab line comment: %s", string(b))
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
