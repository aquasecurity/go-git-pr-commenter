package azure

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

type Azure struct {
	Token    string
	RepoID   string
	PrNumber string
	Project  string
	ApiUrl   string
}

type ThreadsResponse struct {
	Threads []Thread `json:"value,omitempty"`
}

type Thread struct {
	Id       int       `json:"id,omitempty"`
	Comments []Comment `json:"comments,omitempty"`
}

type LineStruct struct {
	Line   int `json:"line,omitempty"`
	Offset int `json:"offset,omitempty"`
}

type ThreadContext struct {
	FilePath       string     `json:"filePath,omitempty"`
	RightFileEnd   LineStruct `json:"rightFileEnd,omitempty"`
	RightFileStart LineStruct `json:"rightFileStart,omitempty"`
}

type Body struct {
	Comments      []Comment     `json:"comments,omitempty"`
	Status        int           `json:"status,omitempty"`
	ThreadContext ThreadContext `json:"threadContext,omitempty"`
}

type Comment struct {
	Id              int    `json:"id,omitempty"`
	ParentCommentId int    `json:"parentCommentId,omitempty"`
	Content         string `json:"content,omitempty"`
}

func NewAzure(token string) (b *Azure, err error) {

	return &Azure{
		Project:  os.Getenv("SYSTEM_TEAMPROJECT"),
		ApiUrl:   os.Getenv("SYSTEM_COLLECTIONURI"),
		Token:    token,
		RepoID:   os.Getenv("BUILD_REPOSITORY_ID"),
		PrNumber: os.Getenv("SYSTEM_PULLREQUEST_PULLREQUESTID"),
	}, nil
}

// WriteMultiLineComment writes a multiline review on a file in the azure PR
func (c *Azure) WriteMultiLineComment(file, comment string, startLine, endLine int) error {

	fmt.Println("initial strat line:", startLine, " end line:", endLine, " file:", file)

	if !strings.HasPrefix(file, "/") {
		file = fmt.Sprintf("/%s", file)
	}

	if startLine == commenter.FIRST_AVAILABLE_LINE {
		startLine = 0
	}

	if endLine == commenter.FIRST_AVAILABLE_LINE {
		endLine = 0
	}

	b := Body{
		Comments: []Comment{
			{
				ParentCommentId: 1,
				Content:         comment,
			},
		},

		Status: 1,
		ThreadContext: ThreadContext{
			FilePath: file,
			RightFileEnd: LineStruct{
				Line:   endLine,
				Offset: 999,
			},
			RightFileStart: LineStruct{
				Line:   startLine,
				Offset: 1,
			},
		},
	}

	reqBody, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("failed to marshal body for azure api: %s", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s/_apis/git/repositories/%s/pullRequests/%s/threads?api-version=6.0",
		c.ApiUrl, c.Project, c.RepoID, c.PrNumber),
		strings.NewReader(string(reqBody)))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth("", c.Token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.Body)
		b, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed write azure line comment: %s", string(b))
	}

	return nil

}

// WriteLineComment writes a single review line on a file of the azure PR
func (c *Azure) WriteLineComment(_, _ string, _ int) error {

	return nil
}

func (c *Azure) RemovePreviousAquaComments(msg string) error {

	resp, err := utils.GetComments(fmt.Sprintf("%s%s/_apis/git/repositories/%s/pullRequests/%s/threads?api-version=6.0",
		c.ApiUrl, c.Project, c.RepoID, c.PrNumber), map[string]string{"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(":"+c.Token))})
	if err != nil {
		return fmt.Errorf("failed getting comments with error: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	commentsResponse := ThreadsResponse{}
	err = json.Unmarshal(body, &commentsResponse)
	if err != nil {
		return fmt.Errorf("failed unmarshal response body with error: %w", err)
	}

	for _, thread := range commentsResponse.Threads {
		for _, comment := range thread.Comments {
			if strings.Contains(comment.Content, msg) {
				err = utils.DeleteComments(fmt.Sprintf("%s%s/_apis/git/repositories/%s/pullRequests/%s/threads/%s/comments/%s?api-version=6.0",
					c.ApiUrl, c.Project, c.RepoID, c.PrNumber, strconv.Itoa(thread.Id), strconv.Itoa(comment.Id)), map[string]string{"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(":"+c.Token))})
				if err != nil {
					return fmt.Errorf("failed deleting comment with error: %w", err)
				}
			}
		}
	}
	return nil
}
