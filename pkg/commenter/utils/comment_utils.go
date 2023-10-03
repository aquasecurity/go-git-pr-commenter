package utils

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

func UrlWithParams(baseUrl string, params map[string]string) (string, error) {
	newUrl, err := url.Parse(baseUrl)
	if err != nil {
		return "", err
	}
	q := newUrl.Query()
	for key, value := range params {
		q.Add(key, value)
	}
	newUrl.RawQuery = q.Encode()
	return newUrl.String(), nil
}

func DeleteComments(url string, headers map[string]string) error {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	_, err = client.Do(req)
	if err != nil {
		return err
	}
	return nil
}

func GetComments(url string, headers map[string]string) (*http.Response, error) {
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	return client.Do(req)
}

func GetRepositoryCloneURL() (string, error) {
	if cloneUrl, isExist := os.LookupEnv("GIT_URL"); isExist {
		fmt.Println("Using GIT_URL env var as clone url: ", cloneUrl)
		return cloneUrl, nil
	}

	return getGitRemoteURL()
}

func getGitRemoteURL() (string, error) {
	repositoryPath, ok := os.LookupEnv("WORKSPACE")
	if !ok {
		return "", errors.New("could not find remote url, no WORKSPACE env var")
	}
	remotes, err := getGitRemotes(repositoryPath)
	if err != nil {
		return "", fmt.Errorf("failed to get git remotes: %w", err)
	}

	if len(remotes) == 0 {
		return "", errors.New("no git remotes found")
	}

	remoteUrl := remotes[0][1]

	for _, remote := range remotes {
		if remote[0] == "origin" {
			remoteUrl = remote[1]
			break
		}
	}

	fmt.Println("Using remote url extracted from WORKSPACE: ", remoteUrl)

	return remoteUrl, nil
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
