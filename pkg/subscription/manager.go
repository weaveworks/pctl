package subscription

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Manager manages getting and list profile subscriptions
type Manager struct {
	kClient client.Client
	ctx     context.Context
}

// NewManager returns a SubscriptionManager
func NewManager(kClient client.Client) *Manager {
	return &Manager{
		kClient: kClient,
		ctx:     context.TODO(),
	}
}
