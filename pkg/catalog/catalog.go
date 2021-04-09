package catalog

import (
	"net/http"
	"net/url"
)

//go:generate counterfeiter -o fakes/fake_http_client.go . HTTPClient
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

var httpClient HTTPClient = http.DefaultClient

func doRequest(u *url.URL, q url.Values) (*http.Response, error) {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	if q != nil {
		req.URL.RawQuery = q.Encode()
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
