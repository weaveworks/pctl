package catalog

import (
	"fmt"

	"github.com/weaveworks/pctl/pkg/subscription"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ProfileData data containing profile and available version update for format printing.
type ProfileData struct {
	Profile                 subscription.SubscriptionSummary
	AvailableVersionUpdates []string
}

// List will fetch all installed profiles on the cluster and check if there are updated versions available.
func List(k8sClient runtimeclient.Client, catalogClient CatalogClient) ([]ProfileData, error) {
	profiles, err := subscription.NewManager(k8sClient).List()
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		fmt.Println("no profiles found")
		return nil, nil
	}
	profileData := make([]ProfileData, 0)
	for _, p := range profiles {
		var versions []string
		related, err := GetAvailableUpdates(catalogClient, p.Catalog, p.Profile, p.Version)
		if err != nil {
			return nil, fmt.Errorf("failed to search for profile %s for updates: %w", p.Profile, err)
		}
		for _, r := range related {
			versions = append(versions, r.Version)
		}
		profileData = append(profileData, ProfileData{
			Profile:                 p,
			AvailableVersionUpdates: versions,
		})
	}
	return profileData, nil
}
