package azure

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type Azure struct {
	Token    string
	RepoID   string
	PrNumber string
	Project  string
	ApiUrl   string
}

type LineStruct struct {
	Line   int `json:"line,omitempty"`
	Offset int `json:"offset,omitempty"`
}

type ThreadContext struct {
	FilePath      string     `json:"filePath,omitempty"`
	LeftFileEnd   LineStruct `json:"leftFileEnd,omitempty"`
	LeftFileStart LineStruct `json:"leftFileStart,omitempty"`
}

type Body struct {
	Comments      []Comment     `json:"comments,omitempty"`
	Status        int           `json:"status,omitempty"`
	ThreadContext ThreadContext `json:"threadContext,omitempty"`
}

type Comment struct {
	ParentCommentId int    `json:"parentCommentId,omitempty"`
	Content         string `json:"content,omitempty"`
	CommentType     int    `json:"commentType,omitempty"`
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

	if !strings.HasPrefix(file, "/") {
		file = fmt.Sprintf("/%s", file)
	}

	b := Body{
		Comments: []Comment{
			{
				ParentCommentId: 1,
				Content:         comment,
				CommentType:     1,
			},
		},

		Status: 1,
		ThreadContext: ThreadContext{
			FilePath: file,
			LeftFileEnd: LineStruct{
				Line:   endLine,
				Offset: 0,
			},
			LeftFileStart: LineStruct{
				Line:   startLine,
				Offset: 0,
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
		b, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed write azure line comment: %s", string(b))
	}

	return nil

}

// WriteLineComment writes a single review line on a file of the azure PR
func (c *Azure) WriteLineComment(_, _ string, _ int) error {

	return nil
}
