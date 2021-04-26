package profile

import (
	"context"
	"fmt"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SubscriptionDescription contains a description of a subscription
type SubscriptionDescription struct {
	Name      string
	Namespace string
	Ready     string
}

// SubscriptionManager manages getting and list profile subscriptions
type SubscriptionManager struct {
	kClient client.Client
	ctx     context.Context
}

// New returns a SubscriptionManager
func New(kClient client.Client) *SubscriptionManager {
	return &SubscriptionManager{
		kClient: kClient,
		ctx:     context.TODO(),
	}
}

// List returns a list of subscriptions
func (sm *SubscriptionManager) List() ([]SubscriptionDescription, error) {
	var subscriptions profilesv1.ProfileSubscriptionList
	err := sm.kClient.List(sm.ctx, &subscriptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list profile subscriptions: %w", err)
	}
	var descriptions []SubscriptionDescription
	for _, sub := range subscriptions.Items {
		status := "Unknown"
		for _, cond := range sub.Status.Conditions {
			if cond.Type == "Ready" {
				status = string(sub.Status.Conditions[0].Status)
				break
			}
		}

		descriptions = append(descriptions, SubscriptionDescription{
			Name:      sub.Name,
			Namespace: sub.Namespace,
			Ready:     status,
		})
	}
	return descriptions, nil
}
