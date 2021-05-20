package subscription

import (
	"fmt"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

// List returns a list of subscriptions
func (sm *Manager) List() ([]SubscriptionSummary, error) {
	var subscriptions profilesv1.ProfileSubscriptionList
	err := sm.kClient.List(sm.ctx, &subscriptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list profile subscriptions: %w", err)
	}
	var descriptions []SubscriptionSummary
	for _, sub := range subscriptions.Items {
		version := "-"
		profile := "-"
		catalog := "-"
		if sub.Spec.ProfileCatalogDescription != nil {
			version = sub.Spec.ProfileCatalogDescription.Version
			profile = sub.Spec.ProfileCatalogDescription.Profile
			catalog = sub.Spec.ProfileCatalogDescription.Catalog
		}
		descriptions = append(descriptions, SubscriptionSummary{
			Name:      sub.Name,
			Namespace: sub.Namespace,
			Version:   version,
			Profile:   profile,
			Catalog:   catalog,
		})
	}
	return descriptions, nil
}
