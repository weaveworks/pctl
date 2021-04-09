package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/weaveworks/profiles/api/v1alpha1"
)

// Search queries the catalog at catalogURL for profiles matching the provided searchName.
func Search(catalogURL, searchName string) ([]v1alpha1.ProfileDescription, error) {
	u, err := url.Parse(catalogURL)
	if err != nil {
		return []v1alpha1.ProfileDescription{}, fmt.Errorf("failed to parse url %q: %w", catalogURL, err)
	}
	u.Path = "profiles"
	q := u.Query()
	q.Add("name", searchName)
	resp, err := doRequest(u, q)
	if err != nil {
		return []v1alpha1.ProfileDescription{}, fmt.Errorf("failed to do request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Failed to close the response body from profile search with error: %v/n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return []v1alpha1.ProfileDescription{}, fmt.Errorf("failed to fetch catalog: status code %d", resp.StatusCode)
	}

	profiles := []v1alpha1.ProfileDescription{}
	err = json.NewDecoder(resp.Body).Decode(&profiles)
	if err != nil {
		return []v1alpha1.ProfileDescription{}, fmt.Errorf("failed to parse catalog: %w", err)
	}

	if len(profiles) == 0 {
		return []v1alpha1.ProfileDescription{}, fmt.Errorf("no profiles matching %q found", searchName)
	}

	return profiles, nil
}
