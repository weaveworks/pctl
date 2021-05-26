package catalog

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-version"
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
		related, err := Search(catalogClient, p.Profile)
		if err != nil {
			return nil, fmt.Errorf("failed to search for profile %s for updates: %w", p.Profile, err)
		}
		pv, err := version.NewVersion(strings.TrimPrefix(p.Version, "v"))
		if err != nil {
			return nil, fmt.Errorf("failed to format profile %s with version %s into version: %w", p.Profile, p.Version, err)
		}
		for _, r := range related {
			// Search uses Contains, which will match too many things. We want an exact match.
			// TODO: Maybe delegate this to the profile catalog?
			if r.Name != p.Profile {
				continue
			}
			rv, err := version.NewVersion(strings.TrimPrefix(r.Version, "v"))
			if err != nil {
				return nil, fmt.Errorf("failed to format profile %s with version %s into version: %w", r.Name, r.Version, err)
			}
			if rv.GreaterThan(pv) {
				versions = append(versions, "v"+rv.String())
			}
		}
		profileData = append(profileData, ProfileData{
			Profile:                 p,
			AvailableVersionUpdates: versions,
		})
	}
	return profileData, nil
}
