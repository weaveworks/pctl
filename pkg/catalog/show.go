package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/weaveworks/profiles/api/v1alpha1"
)

func Show(catalogURL, profileName string) (v1alpha1.ProfileDescription, error) {
	u, err := url.Parse(catalogURL)
	if err != nil {
		return v1alpha1.ProfileDescription{}, fmt.Errorf("failed to parse url %q: %w", catalogURL, err)
	}
	u.Path = "profiles/" + profileName
	resp, err := doRequest(u, nil)
	if err != nil {
		return v1alpha1.ProfileDescription{}, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return v1alpha1.ProfileDescription{}, fmt.Errorf("unable to find profile `%s` in catalog %s", profileName, catalogURL)
	}

	if resp.StatusCode != http.StatusOK {
		return v1alpha1.ProfileDescription{}, fmt.Errorf("failed to fetch profile: status code %d", resp.StatusCode)
	}

	profile := v1alpha1.ProfileDescription{}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return v1alpha1.ProfileDescription{}, fmt.Errorf("failed to parse profile: %w", err)
	}

	return profile, nil
}
