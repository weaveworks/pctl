package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// HTTPClient defines an http client which then can be used to test the
// handler code.
//go:generate counterfeiter -o fakes/fake_http_client.go . HTTPClient
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

var httpClient HTTPClient = http.DefaultClient

// Search queries the catalog at catalogURL for profiles matching the provided profileName.
func Search(catalogURL, searchName string) ([]ProfileDescription, error) {
	u, err := url.Parse(catalogURL)
	if err != nil {
		return []ProfileDescription{}, fmt.Errorf("failed to parse url %q: %w", catalogURL, err)
	}
	u.Path = "profiles"
	q := u.Query()
	q.Add("name", searchName)
	resp, err := doRequest(u, q)
	if err != nil {
		return []ProfileDescription{}, fmt.Errorf("failed to do request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Failed to close the response body from profile search with error: %v/n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return []ProfileDescription{}, fmt.Errorf("failed to fetch catalog: status code %d", resp.StatusCode)
	}

	profiles := []ProfileDescription{}
	err = json.NewDecoder(resp.Body).Decode(&profiles)
	if err != nil {
		return []ProfileDescription{}, fmt.Errorf("failed to parse catalog: %w", err)
	}

	if len(profiles) == 0 {
		return []ProfileDescription{}, fmt.Errorf("no profiles matching %q found", searchName)
	}

	return profiles, nil
}
