package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

// Show queries the catalog at catalogURL for a profile matching the provided profileName
func Show(catalogClient CatalogClient, catalogName, profileName, catalogVersion string) (profilesv1.ProfileDescription, error) {
	u, err := url.Parse("/profiles")
	if err != nil {
		return profilesv1.ProfileDescription{}, err
	}
	u.Path = path.Join(u.Path, catalogName, profileName, catalogVersion)
	data, code, err := catalogClient.DoRequest(u.String(), nil)
	if err != nil {
		return profilesv1.ProfileDescription{}, fmt.Errorf("failed to do request: %w", err)
	}

	if code != http.StatusOK {
		if code == http.StatusNotFound {
			return profilesv1.ProfileDescription{},
				fmt.Errorf("unable to find profile %q in catalog %q (with version if provided: %s)",
					profileName, catalogName, catalogVersion)
		}
		return profilesv1.ProfileDescription{}, fmt.Errorf("failed to fetch profile from catalog, status code %d", code)
	}

	var profile profilesv1.ProfileDescription
	if err := json.Unmarshal(data, &profile); err != nil {
		return profilesv1.ProfileDescription{}, fmt.Errorf("failed to parse profile: %w", err)
	}

	return profile, nil
}
