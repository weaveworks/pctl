package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func Show(catalogURL, profileName string) (ProfileDescription, error) {
	u, err := url.Parse(catalogURL)
	if err != nil {
		return ProfileDescription{}, fmt.Errorf("failed to parse url %q: %w", catalogURL, err)
	}
	u.Path = "profiles/" + profileName
	resp, err := doRequest(u, nil)
	if err != nil {
		return ProfileDescription{}, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ProfileDescription{}, fmt.Errorf("unable to find profile `%s` in catalog %s", profileName, catalogURL)
	}

	if resp.StatusCode != http.StatusOK {
		return ProfileDescription{}, fmt.Errorf("failed to fetch profile: status code %d", resp.StatusCode)
	}

	profile := ProfileDescription{}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return ProfileDescription{}, fmt.Errorf("failed to parse profile: %w", err)
	}

	return profile, nil
}
