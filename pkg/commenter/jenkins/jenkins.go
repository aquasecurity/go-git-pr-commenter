package jenkins

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter"
	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/bitbucket"
	bitbucket_server "github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/bitbucket-server"
	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/utils"
	bitbucketutils "github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/utils/bitbucket"
)

func NewJenkins(baseRef string) (commenter.Repository, error) {
	cloneUrl, _ := utils.GetRepositoryCloneURL()
	fmt.Printf("clone url is %s", cloneUrl)
	_, r := bitbucketutils.GetBitbucketPayload()
	fmt.Printf("bitbucketutils.GetBitbucketPayload() returns %b", r)

	if _, exists := bitbucketutils.GetBitbucketPayload(); strings.Contains(cloneUrl, "bitbucket") || exists {
		username, ok := os.LookupEnv("USERNAME")
		if !ok {
			return nil, fmt.Errorf("USERNAME env var is not set")
		}
		password, ok := os.LookupEnv("PASSWORD")
		if !ok {
			return nil, fmt.Errorf("PASSWORD env var is not set")
		}

		if strings.Contains(cloneUrl, "bitbucket.org") {
			return bitbucket.CreateClient(username, password, bitbucketutils.GetPrId(), bitbucketutils.GetRepositoryName(cloneUrl))
		} else { // bitbucket server
			repoName := bitbucketutils.GetRepositoryName(cloneUrl)
			project, repo := bitbucketutils.GetProjectAndRepo(repoName)
			apiUrl, err := getBaseUrl(cloneUrl)
			if err != nil {
				return nil, err
			}
			return bitbucket_server.NewBitbucketServer(apiUrl, username, password, bitbucketutils.GetPrId(), project, repo, baseRef)
		}
	} else if strings.Contains(cloneUrl, "github") {

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
