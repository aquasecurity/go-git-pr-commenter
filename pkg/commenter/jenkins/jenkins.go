package jenkins

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter"
	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/bitbucket"
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

func getRepositoryCloneURL(repositoryPath string) (string, error) {
	if cloneUrl, isExist := os.LookupEnv("GIT_URL"); isExist {
		return cloneUrl, nil
	}
	return GetGitRemoteURL(repositoryPath)
}

func GetGitRemoteURL(repositoryPath string) (string, error) {
	remotes, err := getGitRemotes(repositoryPath)
	if err != nil {
		return "", err
	}

	if len(remotes) == 0 {
		return "", errors.New("No git remotes found")
	}

	for _, remote := range remotes {
		if remote[0] == "origin" {
			return remote[1], nil
		}
	}

	return remotes[0][1], nil
}

func getGitRemotes(repositoryPath string) ([][]string, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, errors.New("git not found")
	}

	cmd := exec.Command(gitPath, "remote", "-v")
	cmd.Dir = repositoryPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf(`failed to execute git command "%s" - %s - %s`, cmd.String(), output, err.Error())
	}

	outputAsString := string(output)
	outputAsString = strings.TrimSuffix(outputAsString, "\n")
	lines := strings.Split(outputAsString, "\n")
	remotes := [][]string{}
	for _, line := range lines {
		remotes = append(remotes, strings.Fields(line))
	}
	return remotes, nil
}

func NewJenkins() (commenter.Repository, error) {
	repositoryPath := os.Getenv("WORKSPACE")
	cloneUrl, _ := getRepositoryCloneURL(repositoryPath)

	if strings.Contains(cloneUrl, "bitbucket.org") {
		payload := &bitbucketPayload{}
		err := json.Unmarshal([]byte(os.Getenv("BITBUCKET_PAYLOAD")), payload)
		if err != nil {
			return nil, err
		}
		return bitbucket.CreateClient(os.Getenv("USERNAME"), os.Getenv("PASSWORD"), os.Getenv("BITBUCKET_PULL_REQUEST_ID"), payload.Repository.Owner.DisplayName+"/"+payload.Repository.Name)
	}

	return nil, nil
}
