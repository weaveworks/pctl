package catalog

import (
	"fmt"
	"net/http"
	"strings"

	"gopkg.in/yaml.v2"
)

//go:generate counterfeiter -o fakes/fake_http_client.go . HTTPClient
type HTTPClient interface {
	Get(string) (*http.Response, error)
}

var httpClient HTTPClient = http.DefaultClient

func Search(catalogURL, profileName string) ([]ProfileDescription, error) {
	resp, err := httpClient.Get(catalogURL)
	if err != nil {
		return []ProfileDescription{}, fmt.Errorf("failed to fetch catalog: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return []ProfileDescription{}, fmt.Errorf("failed to fetch catalog: status code %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	profiles := ProfileCatalog{}
	err = yaml.NewDecoder(resp.Body).Decode(&profiles)
	if err != nil {
		return []ProfileDescription{}, fmt.Errorf("failed to parse catalog: %w", err)
	}

	var profileDescriptions []ProfileDescription
	for _, profile := range profiles.Spec.Profiles {
		if strings.Contains(profile.Name, profileName) {
			profileDescriptions = append(profileDescriptions, profile)
		}
	}

	if len(profileDescriptions) == 0 {
		return []ProfileDescription{}, fmt.Errorf("no profiles matching %q found", profileName)
	}

	return profileDescriptions, nil
}
