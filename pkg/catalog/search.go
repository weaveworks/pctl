package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

//go:generate counterfeiter -o fakes/fake_http_client.go . HTTPClient
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

var httpClient HTTPClient = http.DefaultClient

func Search(catalogURL, profileName string) ([]ProfileDescription, error) {
	u, err := url.Parse(catalogURL)
	if err != nil {
		return []ProfileDescription{}, fmt.Errorf("failed to parse url %q: %w", catalogURL, err)
	}
	u.Path = "profiles"
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return []ProfileDescription{}, fmt.Errorf("failed to build request: %w", err)
	}
	q := u.Query()
	q.Add("name", profileName)
	req.URL.RawQuery = q.Encode()
	resp, err := httpClient.Do(req)
	if err != nil {
		return []ProfileDescription{}, fmt.Errorf("failed to fetch catalog: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return []ProfileDescription{}, fmt.Errorf("failed to fetch catalog: status code %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	profiles := []ProfileDescription{}
	err = json.NewDecoder(resp.Body).Decode(&profiles)
	if err != nil {
		return []ProfileDescription{}, fmt.Errorf("failed to parse catalog: %w", err)
	}

	if len(profiles) == 0 {
		return []ProfileDescription{}, fmt.Errorf("no profiles matching %q found", profileName)
	}

	return profiles, nil
}
