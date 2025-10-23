package bitbucket_server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/utils"
	change_report "github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/utils/change-report"

	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter"
)

const LIMIT = 500

type BitbucketServer struct {
	Token        string
	UserName     string
	Project      string
	Repo         string
	PrNumber     string
	ApiUrl       string
	ChangeReport change_report.ChangeReport
}

type ActivitiesResponse struct {
	Activities    []Activity `json:"values,omitempty"`
	IsLastPage    bool       `json:"isLastPage"`
	Start         int        `json:"start"`
	NextPageStart int        `json:"nextPageStart,omitempty"`
}

type Activity struct {
	Id            int     `json:"id,omitempty"`
	Action        string  `json:"action,omitempty"`
	CommentAction string  `json:"commentAction,omitempty"`
	Comment       Comment `json:"comment,omitempty"`
}

type Comment struct {
	Id      int    `json:"id,omitempty"`
	Version int    `json:"version,omitempty"`
	Text    string `json:"text,omitempty"`
}

type NewComment struct {
	Test   string `json:"text"`
	Anchor Anchor `json:"anchor"`
}

type Anchor struct {
	Line     int    `json:"line"`
	LineType string `json:"lineType"`
	FileType string `json:"fileType"`
	Path     string `json:"path"`
}

func NewBitbucketServer(apiUrl, userName, token, prNumber, project, repo, baseRef string) (b *BitbucketServer, err error) {
	fmt.Println("23/10 - NewBitbucketServer")
	changeReport, err := change_report.GenerateChangeReport(baseRef)
	fmt.Println("Creating Bitbucket Server client parameters:")
	fmt.Println("apiUrl: ", apiUrl)
	fmt.Println("userName: ", userName)
	fmt.Println("prNumber: ", prNumber)
	fmt.Println("project: ", project)
	fmt.Println("repo: ", repo)

	return &BitbucketServer{
		ApiUrl:       apiUrl,
		UserName:     userName,
		Token:        token,
		Project:      project,
		Repo:         repo,
		PrNumber:     prNumber,
		ChangeReport: changeReport,
	}, err
}

func (c *BitbucketServer) WriteMultiLineComment(file, comment string, startLine, _ int) error {
	// In bitbucket we support one line only
	err := c.WriteLineComment(file, comment, startLine)
	if err != nil {
		return fmt.Errorf("failed to write bitbucket server multiline comment: %w", err)
	}

	return nil
}

func (c *BitbucketServer) WriteLineComment(file, comment string, line int) error {
	if line == commenter.FIRST_AVAILABLE_LINE || line == 0 {
		line = 1
	}

	changeType := change_report.CONTEXT
	if filechange, ok := c.ChangeReport[file]; ok {
		if _, ok := filechange.AddedLines[line]; ok {
			changeType = change_report.ADDED
		}
	}

	b := NewComment{
		Test: comment,
		Anchor: Anchor{
			Line:     line,
			LineType: string(changeType),
			FileType: "TO",
			Path:     file,
		},
	}

	reqBody, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("failed to marshal body for bitbucket server api: %s", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", c.getCommentPostUrl(), strings.NewReader(string(reqBody)))
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
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed write bitbucket line comment: %s", string(b))
	}

	return nil
}

func (c *BitbucketServer) getIdsToRemove(commentsToRemove []Comment, msg string, start int) ([]Comment, error) {
	fmt.Println("23/10 - getIdsToRemove")
	url, err := utils.UrlWithParams(c.getCommentsUrl(), getCommentsParams(start))
	if err != nil {
		return nil, fmt.Errorf("failed to create comments url: %w", err)
	}
	fmt.Println("23/10 - url: ", url)
	resp, err := utils.GetComments(url, c.getAuthHeaders())

	if err != nil {
		return nil, fmt.Errorf("failed getting comments with error: %w", err)
	}
	fmt.Println("23/10 - resp.Status: ", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	activitiesResponse := ActivitiesResponse{}
	err = json.Unmarshal(body, &activitiesResponse)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshal response body with error: %w", err)
	}

	for _, value := range activitiesResponse.Activities {
		if value.CommentAction == "ADDED" && value.Action == "COMMENTED" && strings.Contains(value.Comment.Text, msg) {
			commentsToRemove = append(commentsToRemove, value.Comment)
		}
	}

	if activitiesResponse.IsLastPage {
		return commentsToRemove, nil
	}
	return c.getIdsToRemove(commentsToRemove, msg, activitiesResponse.NextPageStart)

}

func (c *BitbucketServer) RemovePreviousAquaComments(msg string) error {
	fmt.Println("23/10 - RemovePreviousAquaComments")
	var commentsToRemove []Comment
	commentsToRemove, err := c.getIdsToRemove(commentsToRemove, msg, 0)
	if err != nil {
		return err
	}
	fmt.Println("23/10 - len commentsToRemove: ", len(commentsToRemove))
	for _, comment := range commentsToRemove {
		url, _ := utils.UrlWithParams(c.getCommentDeleteUrl(comment.Id), map[string]string{"version": strconv.Itoa(comment.Version)})
		err := utils.DeleteComments(url, c.getAuthHeaders())
		if err != nil {
			fmt.Printf("failed to delete comment with id %d: %s", comment.Id, err)
		}
	}

	return nil
}

func (c *BitbucketServer) getCommentsUrl() string {
	return fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/pull-requests/%s/activities", c.ApiUrl, c.Project, c.Repo, c.PrNumber)
}

func (c *BitbucketServer) getCommentDeleteUrl(id int) string {
	return fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/pull-requests/%s/comments/%d", c.ApiUrl, c.Project, c.Repo, c.PrNumber, id)
}

func (c *BitbucketServer) getCommentPostUrl() string {
	return fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/pull-requests/%s/comments", c.ApiUrl, c.Project, c.Repo, c.PrNumber)
}

func (c *BitbucketServer) getAuthHeaders() map[string]string {
	userToken := []byte(fmt.Sprintf("%s:%s", c.UserName, c.Token))
	return map[string]string{
		"Authorization": fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString(userToken)),
	}
}

func getCommentsParams(start int) map[string]string {
	return map[string]string{
		"start": strconv.Itoa(start),
		"limit": strconv.Itoa(LIMIT),
	}
}
