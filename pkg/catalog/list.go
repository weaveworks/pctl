package catalog

import (
	"fmt"
	"strings"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/weaveworks/pctl/pkg/installation"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

// ProfileData data containing profile and available version update for format printing.
type ProfileData struct {
	Profile                 installation.Summary
	AvailableVersionUpdates []string
}

// List will fetch all installed profiles on the cluster and check if there are updated versions available.
func (m *Manager) List(k8sClient runtimeclient.Client, catalogClient CatalogClient, name string) ([]ProfileData, error) {
	profiles, err := installation.NewManager(k8sClient).List()
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, nil
	}
	profileData := make([]ProfileData, 0)
	for _, p := range profiles {
		var versions []string
		// skip for profiles which don't have a catalog entry. i.e.: profiles installed via branch, url, path.
		if p.Catalog != "-" {
			related, err := GetAvailableUpdates(catalogClient, p.Catalog, p.Profile, p.Version)
			if err != nil {
				return nil, fmt.Errorf("failed to get available updates: %w", err)
			}
			for _, r := range related {
				versions = append(versions, profilesv1.GetVersionFromTag(r.Tag))
			}
		}
		if len(versions) == 0 {
			versions = append(versions, "-")
		}

		// filter results if name is provided
		if name != "" {
			if strings.Contains(p.Name, name) {
				profileData = append(profileData, ProfileData{
					Profile:                 p,
					AvailableVersionUpdates: versions,
				})
			}
		} else {
			profileData = append(profileData, ProfileData{
				Profile:                 p,
				AvailableVersionUpdates: versions,
			})
		}

	}
	return profileData, nil
}
