package utils

import (
	"net/http"
)

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
