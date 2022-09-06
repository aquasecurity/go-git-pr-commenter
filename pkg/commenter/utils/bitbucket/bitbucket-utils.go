package utils

import (
	"encoding/json"
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
		return id
	}

	if id, exists := os.LookupEnv("CHANGE_ID"); exists {
		return id
	}

	return ""
}

func GetRepositoryName() string {
	payload, exists := GetBitbucketPayload()
	if exists {
		return payload.Repository.Owner.DisplayName + "/" + payload.Repository.Name
	}

	if url, exists := os.LookupEnv("GIT_URL"); exists {
		nameRegexp := regexp.MustCompile(`(([^\/]+)\/([^\/]+))(?:\.git)$`)
		name := nameRegexp.FindStringSubmatch(url)[1]
		return name
	}

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
