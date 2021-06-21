package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

// GetAvailableUpdates queries the catalog at catalogURL for profiles which have greater versions than the current
// given one.
func GetAvailableUpdates(catalogClient CatalogClient, catalogName, profileName, profileVersion string) ([]profilesv1.ProfileCatalogEntry, error) {
	u, err := url.Parse("/profiles")
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, catalogName, profileName, profileVersion, "available_updates")
	data, statusCode, err := catalogClient.DoRequest(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog: %w", err)
	}

	if statusCode != http.StatusOK {
		if statusCode == http.StatusNotFound {
			// 404 right now is valid in case there are no updated versions
			// this must be re-visited after https://github.com/weaveworks/profiles/issues/143
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch available updates for profile, status code %d", statusCode)
	}
	var profiles []profilesv1.ProfileCatalogEntry
	if err := json.Unmarshal(data, &profiles); err != nil {
		return nil, fmt.Errorf("failed to parse catalog: %w", err)
	}

	return profiles, nil
}
