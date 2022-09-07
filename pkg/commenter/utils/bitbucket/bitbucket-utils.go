package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type bitbucketRepositoryOwnerPyaload struct {
	DisplayName string `json:"display_name"`
}

type bitbucketRepositoryPayload struct {
	Name  string
	Owner bitbucketRepositoryOwnerPyaload
}

type bitbucketPayload struct {
	Repository bitbucketRepositoryPayload
}

func GetPrId() string {
	if id, exists := os.LookupEnv("BITBUCKET_PULL_REQUEST_ID"); exists {
		fmt.Println("Using pull request id from BITBUCKET_PULL_REQUEST_ID: ", id)
		return id
	}

	if id, exists := os.LookupEnv("CHANGE_ID"); exists {
		fmt.Println("Using pull request id from CHANGE_ID: ", id)
		return id
	}

	fmt.Println("Could not find pull request id")
	return ""
}

func GetRepositoryName(cloneUrl string) string {
	payload, exists := GetBitbucketPayload()
	if exists {
		name := payload.Repository.Owner.DisplayName + "/" + payload.Repository.Name
		fmt.Println("Using repository name from BITBUCKET_PAYLOAD: ", name)
		return name
	}

	nameRegexp := regexp.MustCompile(`(([^\/]+)\/([^\/]+))(?:\.git)$`)
	matches := nameRegexp.FindStringSubmatch(cloneUrl)
	if len(matches) > 1 {
		name := nameRegexp.FindStringSubmatch(cloneUrl)[1]
		fmt.Println("Using repository name from cloneUrl: ", name)
		return name
	}

	fmt.Println("Could not find repository name")
	return ""
}

func GetProjectAndRepo(repoName string) (string, string) {
	project, repo := "", ""
	if repoName != "" {
		split := strings.Split(repoName, "/")
		if len(split) == 2 {
			project, repo = split[0], split[1]
		}
	}
	return project, repo
}

func GetBitbucketPayload() (*bitbucketPayload, bool) {
	rawPayload, exists := os.LookupEnv("BITBUCKET_PAYLOAD")
	if !exists {
		return nil, false
	}

	payload := &bitbucketPayload{}
	err := json.Unmarshal([]byte(rawPayload), payload)
	if err != nil {
		return nil, false
	}

	return payload, true
}
