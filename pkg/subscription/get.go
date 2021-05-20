package subscription

import (
	"fmt"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SubscriptionSummary contains a summary of a subscription
type SubscriptionSummary struct {
	Name      string
	Namespace string
	Version   string
	Profile   string
	Catalog   string
}

// Get returns a SubscriptionSummary for a given subscription
func (sm *Manager) Get(namespace, name string) (SubscriptionSummary, error) {
	var sub profilesv1.ProfileSubscription
	var summary SubscriptionSummary
	err := sm.kClient.Get(sm.ctx, client.ObjectKey{Name: name, Namespace: namespace}, &sub)
	if err != nil {
		return SubscriptionSummary{}, fmt.Errorf("failed to get profile subscriptions: %w", err)
	}
	summary = SubscriptionSummary{
		Name:      sub.Name,
		Namespace: sub.Namespace,
	}
	return summary, nil
}
