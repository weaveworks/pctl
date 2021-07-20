package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	"github.com/weaveworks/profiles/pkg/protos"
)

// Show queries the catalog at catalogURL for a profile matching the provided profileName
func (m *Manager) Show(catalogClient CatalogClient, catalogName, profileName, profileVersion string) (profilesv1.ProfileCatalogEntry, error) {
	u, err := url.Parse("/profiles")
	if err != nil {
		return profilesv1.ProfileCatalogEntry{}, err
	}
	u.Path = path.Join(u.Path, catalogName, profileName, profileVersion)
	data, code, err := catalogClient.DoRequest(u.String(), nil)
	if err != nil {
		return profilesv1.ProfileCatalogEntry{}, fmt.Errorf("failed to do request: %w", err)
	}

	if code != http.StatusOK {
		if code == http.StatusNotFound {
			return profilesv1.ProfileCatalogEntry{},
				fmt.Errorf("unable to find profile %q in catalog %q (with version if provided: %s)",
					profileName, catalogName, profileVersion)
		}
		return profilesv1.ProfileCatalogEntry{}, fmt.Errorf("failed to fetch profile from catalog, status code %d", code)
	}

	var profile protos.GRPCProfileCatalogEntry
	if err := json.Unmarshal(data, &profile); err != nil {
		return profilesv1.ProfileCatalogEntry{}, fmt.Errorf("failed to parse profile: %w", err)
	}

	return profile.Item, nil
}
