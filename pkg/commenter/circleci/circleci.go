package circleci

import (
	"fmt"
	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter"
	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/github"
	"github.com/argonsecurity/go-environments/enums"
	"github.com/argonsecurity/go-environments/environments/circleci"
	env_utils "github.com/argonsecurity/go-environments/environments/utils"
	"net/url"
	"os"
)

func NewCircleCi(cloneUrl string, prNumber int) (commenter.Repository, error) {

	scmSource, scmApiUrl := circleci.GetRepositorySource(cloneUrl)

	if scmSource == enums.GithubServer || scmSource == enums.Github {
		_, org, repoName, _ := env_utils.ParseDataFromCloneUrl(cloneUrl, scmApiUrl, scmSource)
		token := os.Getenv("GITHUB_TOKEN")
		//prNumber := os.Getenv("CHANGE_ID")
		//prNumberInt, err := strconv.Atoi(prNumber)
		//if err != nil {
		//	return nil, fmt.Errorf("failed converting prNumber to int, %s: %s", prNumber, err.Error())
		//}

		if scmSource == enums.Github {
			return github.NewGithub(
				token,
				org,
				repoName,
				prNumber)
		} else { //github server
			apiUrl, err := getBaseUrl(cloneUrl)
			if err != nil {
				return nil, err
			}
			return github.NewGithubServer(apiUrl, token, org, repoName, prNumber)
		}

	}

	return nil, nil
}

func getBaseUrl(fullUrl string) (string, error) {
	u, err := url.Parse(fullUrl)
	if err != nil {
		return "", fmt.Errorf("failed to parse url %s - %s", fullUrl, err.Error())
	}

	return fmt.Sprintf("%s://%s", u.Scheme, u.Host), nil
}
