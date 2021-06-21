package installation

import (
	"fmt"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

// Summary contains a summary of a installation
type Summary struct {
	Name      string
	Namespace string
	Version   string
	Profile   string
	Catalog   string
	Branch    string
	Path      string
	URL       string
}

// List returns a list of installations
func (sm *Manager) List() ([]Summary, error) {
	var installations profilesv1.ProfileInstallationList
	err := sm.kClient.List(sm.ctx, &installations)
	if err != nil {
		return nil, fmt.Errorf("failed to list profile installations: %w", err)
	}
	var descriptions []Summary
	for _, sub := range installations.Items {
		version := "-"
		profile := "-"
		catalog := "-"
		branch := "-"
		path := "-"
		url := "-"
		if sub.Spec.Catalog != nil {
			version = sub.Spec.Catalog.Version
			profile = sub.Spec.Catalog.Profile
			catalog = sub.Spec.Catalog.Catalog
		}
		if sub.Spec.Source != nil {
			if sub.Spec.Source.Path != "" {
				path = sub.Spec.Source.Path
			}
			if sub.Spec.Source.Branch != "" {
				branch = sub.Spec.Source.Branch
			}
			if sub.Spec.Source.URL != "" {
				url = sub.Spec.Source.URL
			}
		}
		descriptions = append(descriptions, Summary{
			Name:      sub.Name,
			Namespace: sub.Namespace,
			Version:   version,
			Profile:   profile,
			Catalog:   catalog,
			Branch:    branch,
			Path:      path,
			URL:       url,
		})
	}
	return descriptions, nil
}
