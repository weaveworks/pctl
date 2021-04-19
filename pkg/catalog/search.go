package catalog

import (
	"encoding/json"
	"fmt"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

// Search queries the catalog at catalogURL for profiles matching the provided searchName.
func Search(catalogClient CatalogClient, searchName string) ([]profilesv1.ProfileDescription, error) {
	q := map[string]string{
		"name": searchName,
	}
	data, err := catalogClient.DoRequest("/profiles", q)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog: %w", err)
	}
	var profiles []profilesv1.ProfileDescription
	if err := json.Unmarshal(data, &profiles); err != nil {
		return nil, fmt.Errorf("failed to parse catalog: %w", err)
	}
	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles matching %q found", searchName)
	}

	return profiles, nil
}
