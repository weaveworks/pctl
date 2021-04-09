package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

// Show queries the catalog at catalogURL for a profile matching the provided profileName
func Show(catalogURL, profilePath string) (profilesv1.ProfileDescription, error) {
	u, err := url.Parse(catalogURL)
	if err != nil {
		return profilesv1.ProfileDescription{}, fmt.Errorf("failed to parse url %q: %w", catalogURL, err)
	}

	u.Path = "profiles/" + profilePath
	resp, err := doRequest(u, nil)
	if err != nil {
		return profilesv1.ProfileDescription{}, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		parts := strings.Split(profilePath, "/")
		catalogName, profileName := parts[0], parts[1]
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
