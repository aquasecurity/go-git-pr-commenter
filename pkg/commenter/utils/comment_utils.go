package utils

import (
	"net/http"
	"net/url"
)

func UrlWithParams(baseUrl string, params map[string]string) string {
	newUrl, _ := url.Parse(baseUrl)
	q := newUrl.Query()
	for key, value := range params {
		q.Add(key, value)
	}
	newUrl.RawQuery = q.Encode()
	return newUrl.String()
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
