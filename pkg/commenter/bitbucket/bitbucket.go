package bitbucket

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type Bitbucket struct {
	Token    string
	UserName string
	Repo     string
	PrNumber string
	ApiUrl   string
}

type Body struct {
	Content Content `json:"content,omitempty"`
	Inline  Inline  `json:"inline,omitempty"`
}

type Content struct {
	Raw string `json:"raw,omitempty"`
}

type Inline struct {
	From int    `json:"from,omitempty"`
	To   int    `json:"to,omitempty"`
	Path string `json:"path,omitempty"`
}

func NewBitbucket(userName, token string) (b *Bitbucket, err error) {

	apiUrl := os.Getenv("BITBUCKET_API_URL")
	if apiUrl == "" {
		apiUrl = "https://api.bitbucket.org/2.0/repositories"
	}

	return &Bitbucket{
		ApiUrl:   apiUrl,
		Token:    token,
		UserName: userName,
		PrNumber: os.Getenv("BITBUCKET_PR_ID"),
		Repo:     os.Getenv("BITBUCKET_REPO_FULL_NAME"),
	}, nil
}

// WriteMultiLineComment writes a multiline review on a file in the bitbucket PR
func (c *Bitbucket) WriteMultiLineComment(file, comment string, startLine, _ int) error {
	// In bitbucket we support one line only
	err := c.WriteLineComment(file, comment, startLine)
	if err != nil {
		return fmt.Errorf("failed to write bitbucket multi line comment: %w", err)
	}

	return nil
}

// WriteLineComment writes a single review line on a file of the bitbucket PR
func (c *Bitbucket) WriteLineComment(file, comment string, line int) error {

	b := Body{
		Content: Content{Raw: comment},
		Inline: Inline{
			To:   line,
			Path: file,
		},
	}
	reqBody, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("failed to marshal body for bitbucket api: %s", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s/pullrequests/%s/comments",
		c.ApiUrl, c.Repo, c.PrNumber),
		strings.NewReader(string(reqBody)))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(c.UserName, c.Token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		b, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed write bitbucket line comment: %s", string(b))
	}

	return nil
}
