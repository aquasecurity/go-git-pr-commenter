package azure

import (
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
	Owner    string
	Project  string
}

func NewAzure(token, owner string) (b *Azure, err error) {

	return &Azure{
		Owner:    owner,
		Project:  os.Getenv("SYSTEM_TEAMPROJECT"),
		Token:    token,
		RepoID:   os.Getenv("BUILD_REPOSITORY_ID"),
		PrNumber: os.Getenv("SYSTEM_PULLREQUEST_PULLREQUESTID"),
	}, nil
}

// WriteMultiLineComment writes a multiline review on a file in the azure PR
func (c *Azure) WriteMultiLineComment(file, comment string, startLine, endLine int) error {

	reqBody := strings.NewReader(fmt.Sprintf(`{
    "comments": [{
        "parentCommentId": 0,
        "content":      "%s",
		"commentType":     1
    }],
	"status": 1,
	"threadContext" : {
 		"filePath": "%s",
		"leftFileStart": {
			"line": %d,
			"offset": 1
		},
		"leftFileEnd": {
			"line": %d,
      		"offset": 1
		}
	}}`, comment, file, startLine, endLine))

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/git/repositories/%s/pullRequests/%s/threads?api-version=6.0",
		c.Owner, c.Project, c.RepoID, c.PrNumber),

		reqBody)
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
