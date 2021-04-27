package subscription_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/subscription"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("List", func() {
	var (
		sm              *subscription.Manager
		fakeClient      client.Client
		profileTypeMeta = metav1.TypeMeta{
			Kind:       "ProfileSubscription",
			APIVersion: "weave.works/v1alpha1",
		}
		sub1       = "sub1"
		sub2       = "sub2"
		namespace1 = "namespace1"
		namespace2 = "namespace2"
	)
	BeforeEach(func() {
		scheme := runtime.NewScheme()
		Expect(profilesv1.AddToScheme(scheme)).To(Succeed())
		fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
		pSub1 := &profilesv1.ProfileSubscription{
			TypeMeta: profileTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      sub1,
				Namespace: namespace1,
			},
			Spec: profilesv1.ProfileSubscriptionSpec{
				ProfileURL: "https://github.com/org/repo-name",
				Branch:     "main",
			},
		}
		pSub2 := pSub1.DeepCopy()
		pSub2.Name = sub2
		pSub2.Namespace = namespace2
		Expect(fakeClient.Create(context.TODO(), pSub1)).To(Succeed())
		Expect(fakeClient.Create(context.TODO(), pSub2)).To(Succeed())
		condition := metav1.Condition{
			Type:               "Ready",
			Status:             "True",
			Reason:             "foo",
			LastTransitionTime: metav1.Now(),
		}

		conditions := []metav1.Condition{condition}
		pSub1New := pSub1.DeepCopy()
		pSub1New.Status.Conditions = conditions
		Expect(fakeClient.Status().Patch(context.TODO(), pSub1New, client.MergeFrom(pSub1))).To(Succeed())

		conditions[0].Status = "False"
		pSub2New := pSub2.DeepCopy()
		pSub2New.Status.Conditions = conditions
		Expect(fakeClient.Status().Patch(context.TODO(), pSub2New, client.MergeFrom(pSub2))).To(Succeed())

		sm = subscription.NewManager(fakeClient)
	})

	It("returns a list of profiles deployed in the cluster", func() {
		subs, err := sm.List()
		Expect(err).NotTo(HaveOccurred())
		Expect(subs).To(ConsistOf(
			subscription.SubscriptionDescription{
				Name:      sub1,
				Namespace: namespace1,
				Ready:     "True",
			},
			subscription.SubscriptionDescription{
				Name:      sub2,
				Namespace: namespace2,
				Ready:     "False",
			},
		))
	})

	When("the list fails", func() {
		BeforeEach(func() {
			//remove profilesv1 from scheme
			scheme := runtime.NewScheme()
			fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
			sm = subscription.NewManager(fakeClient)
		})

		It("returns an error", func() {
			_, err := sm.List()
			Expect(err).To(MatchError(ContainSubstring("failed to list profile subscriptions:")))
		})
	})

	When("no ready status condition exists", func() {
		BeforeEach(func() {
			pSub1 := &profilesv1.ProfileSubscription{}
			Expect(fakeClient.Get(context.TODO(), client.ObjectKey{Name: sub1, Namespace: namespace1}, pSub1)).To(Succeed())
			pSub1New := pSub1.DeepCopy()
			pSub1New.Status.Conditions = nil
			Expect(fakeClient.Status().Patch(context.TODO(), pSub1New, client.MergeFrom(pSub1))).To(Succeed())

			pSub2 := &profilesv1.ProfileSubscription{}
			Expect(fakeClient.Get(context.TODO(), client.ObjectKey{Name: sub2, Namespace: namespace2}, pSub2)).To(Succeed())
			pSub2New := pSub2.DeepCopy()
			pSub2New.Status.Conditions[0].Type = "not a ready status"
			Expect(fakeClient.Status().Patch(context.TODO(), pSub2New, client.MergeFrom(pSub2))).To(Succeed())
		})

		It("sets the status to unknown", func() {
			subs, err := sm.List()
			Expect(err).NotTo(HaveOccurred())
			Expect(subs).To(ConsistOf(
				subscription.SubscriptionDescription{
					Name:      sub1,
					Namespace: namespace1,
					Ready:     "Unknown",
				},
				subscription.SubscriptionDescription{
					Name:      sub2,
					Namespace: namespace2,
					Ready:     "Unknown",
				},
			))

		})
	})
})
