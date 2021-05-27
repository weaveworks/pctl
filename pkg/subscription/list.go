package subscription

import (
	"fmt"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

// SubscriptionSummary contains a summary of a subscription
type SubscriptionSummary struct {
	Name      string
	Namespace string
	Version   string
	Profile   string
	Catalog   string
	Branch    string
	Path      string
	URL       string
}

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
		branch := "-"
		path := "-"
		url := "-"
		if sub.Spec.ProfileCatalogDescription != nil {
			version = sub.Spec.ProfileCatalogDescription.Version
			profile = sub.Spec.ProfileCatalogDescription.Profile
			catalog = sub.Spec.ProfileCatalogDescription.Catalog
		}
		if sub.Spec.Path != "" {
			path = sub.Spec.Path
		}
		if sub.Spec.Branch != "" {
			branch = sub.Spec.Branch
		}
		if sub.Spec.ProfileURL != "" {
			url = sub.Spec.ProfileURL
		}
		descriptions = append(descriptions, SubscriptionSummary{
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
