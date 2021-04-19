package catalog

import (
	"encoding/json"
	"fmt"
	"github.com/weaveworks/pctl/pkg/client"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	"net/http"
)

// Show queries the catalog at catalogURL for a profile matching the provided profileName
func Show(catalogClient CatalogClient, catalogName, profileName string) (profilesv1.ProfileDescription, error) {
	data, err := catalogClient.DoRequest(fmt.Sprintf("/profiles/%s/%s", catalogName, profileName), nil)
	if err != nil {
		if se, ok := err.(*client.StatusError); ok {
			if se.Code() == http.StatusNotFound {
				return profilesv1.ProfileDescription{}, fmt.Errorf("unable to find profile `%s` in catalog `%s`", profileName, catalogName)
			}
		}
		return profilesv1.ProfileDescription{}, fmt.Errorf("failed to do request: %w", err)
	}

	var profile profilesv1.ProfileDescription
	if err := json.Unmarshal(data, &profile); err != nil {
		return profilesv1.ProfileDescription{}, fmt.Errorf("failed to parse profile: %w", err)
	}

	return profile, nil
}
