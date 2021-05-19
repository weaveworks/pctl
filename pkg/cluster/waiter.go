package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/apply/poller"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/aggregator"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/collector"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Waiter waits for a set of resources to be Ready.
//go:generate counterfeiter -o fakes/fake_waiter.go . Waiter
type Waiter interface {
	Wait(components ...string) error
}

// KubeConfig defines configurable properties of the kube waiter.
type KubeConfig struct {
	Client    client.Client
	Interval  time.Duration
	Timeout   time.Duration
	Namespace string
}

// KubeWaiter is a kubernetes waiter.
type KubeWaiter struct {
	KubeConfig
	StatusPoller poller.Poller
}

// NewKubeWaiter creates a new KubeWaiter.
func NewKubeWaiter(cfg KubeConfig) *KubeWaiter {
	p := polling.NewStatusPoller(cfg.Client, cfg.Client.RESTMapper())
	return &KubeWaiter{
		KubeConfig:   cfg,
		StatusPoller: p,
	}
}

// Wait waits for some components to be status Ready.
func (w *KubeWaiter) Wait(components ...string) error {
	objects, err := w.buildComponentObjectRefs(components...)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), w.Timeout)
	defer cancel()

	opts := polling.Options{PollInterval: w.Interval, UseCache: true}
	eventsChan := w.StatusPoller.Poll(ctx, objects, opts)

	coll := collector.NewResourceStatusCollector(objects)
	done := coll.ListenWithObserver(eventsChan, desiredStatusNotifierFunc(cancel, status.CurrentStatus))

	<-done

	for _, rs := range coll.ResourceStatuses {
		switch rs.Status {
		case status.CurrentStatus:
			fmt.Printf("%s: %s ready", rs.Identifier.Name, strings.ToLower(rs.Identifier.GroupKind.Kind))
		case status.NotFoundStatus:
			fmt.Printf("%s: %s not found", rs.Identifier.Name, strings.ToLower(rs.Identifier.GroupKind.Kind))
		default:
			fmt.Printf("%s: %s not ready", rs.Identifier.Name, strings.ToLower(rs.Identifier.GroupKind.Kind))
		}
	}

	if coll.Error != nil || ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("timed out waiting for condition")
	}
	return nil
}

func (w *KubeWaiter) buildComponentObjectRefs(components ...string) ([]object.ObjMetadata, error) {
	var objRefs []object.ObjMetadata
	for _, deployment := range components {
		objMeta, err := object.CreateObjMetadata(w.Namespace, deployment, schema.GroupKind{Group: "apps", Kind: "Deployment"})
		if err != nil {
			return nil, err
		}
		objRefs = append(objRefs, objMeta)
	}
	return objRefs, nil
}

// desiredStatusNotifierFunc returns an Observer function for the
// ResourceStatusCollector that will cancel the context (using the cancelFunc)
// when all resources have reached the desired status.
func desiredStatusNotifierFunc(cancelFunc context.CancelFunc,
	desired status.Status) collector.ObserverFunc {
	return func(rsc *collector.ResourceStatusCollector, _ event.Event) {
		var rss []*event.ResourceStatus
		for _, rs := range rsc.ResourceStatuses {
			rss = append(rss, rs)
		}
		aggStatus := aggregator.AggregateStatus(rss, desired)
		if aggStatus == desired {
			cancelFunc()
		}
	}
}
