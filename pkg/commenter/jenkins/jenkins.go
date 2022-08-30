package jenkins

import (
	"encoding/json"
	"os"

	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter"
	"github.com/aquasecurity/go-git-pr-commenter/pkg/commenter/bitbucket"
)

type bitbucketRepositoryPayload struct {
	Name string
}

type bitbucketPayload struct {
	Repository bitbucketRepositoryPayload
}

func NewJenkins() (commenter.Repository, error) {
	if _, exist := os.LookupEnv("BITBUCKET_ACTOR"); exist {
		payload := &bitbucketPayload{}
		err := json.Unmarshal([]byte(os.Getenv("BITBUCKET_PAYLOAD")), payload)
		if err != nil {
			return nil, err
		}
		return bitbucket.CreateClient(os.Getenv("USERNAME"), os.Getenv("PASSWORD"), os.Getenv("BITBUCKET_PULL_REQUEST_ID"), payload.Repository.Name)
	}

	return nil, nil
}
