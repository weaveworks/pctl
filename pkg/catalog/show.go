package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

// Show queries the catalog at catalogURL for a profile matching the provided profileName
func Show(catalogURL, catalogName, profileName string) (profilesv1.ProfileDescription, error) {
	u, err := url.Parse(catalogURL)
	if err != nil {
		return profilesv1.ProfileDescription{}, fmt.Errorf("failed to parse url %q: %w", catalogURL, err)
	}

	u.Path = fmt.Sprintf("profiles/%s/%s", catalogName, profileName)
	resp, err := doRequest(u, nil)
	if err != nil {
		return profilesv1.ProfileDescription{}, fmt.Errorf("failed to do request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close the response body from profile show with error: %v/n", err)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return profilesv1.ProfileDescription{}, fmt.Errorf("unable to find profile `%s` in catalog `%s`", profileName, catalogName)
	}

	if resp.StatusCode != http.StatusOK {
		return profilesv1.ProfileDescription{}, fmt.Errorf("failed to fetch profile: status code %d", resp.StatusCode)
	}

	profile := profilesv1.ProfileDescription{}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return profilesv1.ProfileDescription{}, fmt.Errorf("failed to parse profile: %w", err)
	}

	return profile, nil
}
