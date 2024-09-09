package jenkins

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/github"
	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/gitlab"
	"github.com/rs/zerolog/log"

	"github.com/argonsecurity/go-environments/enums"
	"github.com/argonsecurity/go-environments/environments/jenkins"
	env_utils "github.com/argonsecurity/go-environments/environments/utils"

	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter"
	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/bitbucket"
	bitbucket_server "github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/bitbucket-server"
	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/utils"
	bitbucketutils "github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/utils/bitbucket"
)

func NewJenkins(baseRef string) (commenter.Repository, error) {
	cloneUrl, _ := utils.GetRepositoryCloneURL()
	sanitizedCloneUrl := env_utils.StripCredentialsFromUrl(cloneUrl)
	scmSource, scmApiUrl := jenkins.GetRepositorySource(sanitizedCloneUrl)

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
			return bitbucket_server.NewBitbucketServer(scmApiUrl, username, password, bitbucketutils.GetPrId(), project, repo, baseRef)
		}
	} else if scmSource == enums.GithubServer || scmSource == enums.Github {
		_, org, repoName, _, err := env_utils.ParseDataFromCloneUrl(cloneUrl, scmApiUrl, scmSource)
		if err != nil {
			return nil, fmt.Errorf("failed parsing url with error: %s", err.Error())
		}
		token := os.Getenv("GITHUB_TOKEN")
		prNumber := os.Getenv("CHANGE_ID")
		// for gh single jenkins pipeline
		if prNumber == "" {
			prNumber = os.Getenv("ghprbPullId")
		}
		prNumberInt, err := strconv.Atoi(prNumber)
		if err != nil {
			return nil, fmt.Errorf("failed converting prNumber to int, %s: %s", prNumber, err.Error())
		}

		if scmSource == enums.Github {
			return github.NewGithub(
				token,
				org,
				repoName,
				prNumberInt)
		} else { //github server
			return github.NewGithubServer(scmApiUrl, token, org, repoName, prNumberInt)
		}

	} else if scmSource == enums.GitlabServer || scmSource == enums.Gitlab {
		_, org, repoName, _, err := env_utils.ParseDataFromCloneUrl(cloneUrl, scmApiUrl, scmSource)
		if err != nil {
			return nil, fmt.Errorf("failed parsing url with error: %s", err.Error())
		}
		token := os.Getenv("GITLAB_TOKEN")
		prNumber := os.Getenv("CHANGE_ID")
		gitURL := os.Getenv("GIT_URL")
		if gitURL == "" {
			return nil, fmt.Errorf("GIT_URL env var is not set")
		}
		apiUrl := getGitLabAPIURL(gitURL)
		log.Info().Msgf("apiUrl before: %s", gitURL)
		log.Info().Msgf("apiUrl: %s", apiUrl)
		log.Info().Msgf("org: %s", org)
		log.Info().Msgf("repoName: %s", repoName)
		log.Info().Msgf("prNumber: %s", prNumber)

		return gitlab.NewGitlab(token, apiUrl, fmt.Sprintf("%s/%s", org, repoName), prNumber)
	}

	return nil, nil
}

func getGitLabAPIURL(gitURL string) string {
	// Find the protocol (http:// or https://) position and slice it off
	protocolEnd := strings.Index(gitURL, "//") + 2
	if protocolEnd == 1 {
		return ""
	}

	// Find the position of the first '/' after the domain
	domainEnd := strings.Index(gitURL[protocolEnd:], "/") + protocolEnd
	if domainEnd == -1 {
		return ""
	}

	// Extract the base URL (protocol + domain)
	baseURL := gitURL[:domainEnd]

	// Append the GitLab API path
	return baseURL + "/api/v4"
}
