package cluster

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
)

type fakePoller struct {
	events []event.Event
}

func (f *fakePoller) Poll(ctx context.Context, _ []object.ObjMetadata,
	_ polling.Options) <-chan event.Event {
	eventChannel := make(chan event.Event)
	go func() {
		defer close(eventChannel)
		for _, e := range f.events {
			eventChannel <- e
		}
		<-ctx.Done()
	}()
	return eventChannel
}

var _ = Describe("waiter", func() {
	It("can wait for resources to be active", func() {
		depObject := object.ObjMetadata{
			Name:      "component",
			Namespace: "default",
			GroupKind: schema.GroupKind{
				Group: "apps",
				Kind:  "Deployment",
			},
		}
		p := &fakePoller{
			events: []event.Event{
				{
					EventType: event.ResourceUpdateEvent,
					Resource: &event.ResourceStatus{
						Identifier: depObject,
						Status:     status.CurrentStatus,
						Message:    "current",
					},
				},
			},
		}
		waiter := KubeWaiter{
			KubeConfig: KubeConfig{
				Interval:  1 * time.Second,
				Timeout:   2 * time.Second,
				Namespace: "default",
			},
			StatusPoller: p,
		}
		err := waiter.Wait("component")
		Expect(err).NotTo(HaveOccurred())
	})

	When("the resource doesn't get active in time", func() {
		It("returns a timeout error", func() {
			p := &fakePoller{
				events: []event.Event{},
			}
			waiter := KubeWaiter{
				KubeConfig: KubeConfig{
					Interval:  1 * time.Second,
					Timeout:   2 * time.Second,
					Namespace: "default",
				},
				StatusPoller: p,
			}
			err := waiter.Wait("component")
			Expect(err).To(MatchError("timed out waiting for condition"))
		})
	})
})
