package jenkins

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter"
	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/bitbucket"
	bitbucket_server "github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/bitbucket-server"
	bitbucketutils "github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/utils/bitbucket"
)

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

func NewJenkins(baseRef string) (commenter.Repository, error) {
	repositoryPath := os.Getenv("WORKSPACE")
	cloneUrl, _ := getRepositoryCloneURL(repositoryPath)

	if _, exists := bitbucketutils.GetBitbucketPayload(); strings.Contains(cloneUrl, "bitbucket") || exists {
		username := os.Getenv("USERNAME")
		password := os.Getenv("PASSWORD")

		if strings.Contains(cloneUrl, "bitbucket.org") {
			return bitbucket.CreateClient(username, password, bitbucketutils.GetPrId(), bitbucketutils.GetRepositoryName())
		} else { // bitbucket server
			repoName := bitbucketutils.GetRepositoryName()
			project, repo := bitbucketutils.GetProjectAndRepo(repoName)
			return bitbucket_server.NewBitbucketServer(username, password, bitbucketutils.GetPrId(), project, repo, baseRef)
		}
	}

	return nil, nil
}
