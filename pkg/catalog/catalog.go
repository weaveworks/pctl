package catalog

import (
	"net/http"
	"net/url"
)

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
