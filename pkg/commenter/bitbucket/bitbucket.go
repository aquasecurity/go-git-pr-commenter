package bitbucket

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/utils"

	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter"
)

type Bitbucket struct {
	Token    string
	UserName string
	Repo     string
	PrNumber string
	ApiUrl   string
}
type CommentsResponse struct {
	Values []Value `json:"values,omitempty"`
	Next   string  `json:"next"`
}

type Value struct {
	Id      int     `json:"id,omitempty"`
	Deleted bool    `json:"deleted,omitempty"`
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

func CreateClient(userName, token, prNumber, repoName string) (b *Bitbucket, err error) {

	apiUrl := os.Getenv("BITBUCKET_API_URL")
	if apiUrl == "" {
		apiUrl = "https://api.bitbucket.org/2.0/repositories"
	}

	return &Bitbucket{
		ApiUrl:   apiUrl,
		Token:    token,
		UserName: userName,
		PrNumber: prNumber,
		Repo:     repoName,
	}, nil
}

func NewBitbucket(userName, token string) (b *Bitbucket, err error) {
	return CreateClient(userName, token, os.Getenv("BITBUCKET_PR_ID"), os.Getenv("BITBUCKET_REPO_FULL_NAME"))
}

// WriteMultiLineComment writes a multiline review on a file in the bitbucket PR
func (c *Bitbucket) WriteMultiLineComment(file, comment string, startLine, _ int) error {
	// In bitbucket we support one line only
	fmt.Printf("file - %s - comment -%s- starLinr - %s", file, comment, startLine)
	err := c.WriteLineComment(file, comment, startLine)
	if err != nil {
		return fmt.Errorf("failed to write bitbucket multi line comment: %w", err)
	}

	return nil
}

// WriteLineComment writes a single review line on a file of the bitbucket PR
func (c *Bitbucket) WriteLineComment(file, comment string, line int) error {
	if line == commenter.FIRST_AVAILABLE_LINE {
		line = 1
	}
	b := Value{
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

	fmt.Printf("REQ %s, Auth %s", req.URL, req.Header["Authorization"])

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

func (c *Bitbucket) getIdsToRemove(commentIdsToRemove []int, msg string, url string) ([]int, error) {
	resp, err := utils.GetComments(url, map[string]string{"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(c.UserName+":"+c.Token))})
	if err != nil {
		return nil, fmt.Errorf("failed getting comments with error: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	commentsResponse := CommentsResponse{}
	err = json.Unmarshal(body, &commentsResponse)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshal response body with error: %w", err)
	}

	for _, value := range commentsResponse.Values {
		if !value.Deleted && strings.Contains(value.Content.Raw, msg) {
			commentIdsToRemove = append(commentIdsToRemove, value.Id)
		}
	}

	if commentsResponse.Next == "" {
		return commentIdsToRemove, nil
	}
	return c.getIdsToRemove(commentIdsToRemove, msg, commentsResponse.Next)

}

func (c *Bitbucket) RemovePreviousAquaComments(msg string) error {
	var commentIdsToRemove []int
	commentIdsToRemove, err := c.getIdsToRemove(commentIdsToRemove,
		msg, fmt.Sprintf("%s/%s/pullrequests/%s/comments",
			c.ApiUrl,
			c.Repo,
			c.PrNumber))
	if err != nil {
		return err
	}

	for _, commentId := range commentIdsToRemove {
		err = utils.DeleteComments(
			fmt.Sprintf("%s/%s/pullrequests/%s/comments/%s",
				c.ApiUrl,
				c.Repo,
				c.PrNumber,
				strconv.Itoa(commentId)),
			map[string]string{"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(c.UserName+":"+c.Token))})
		if err != nil {
			return err
		}
	}
	return nil
}
