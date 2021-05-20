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

var _ = Describe("Get", func() {
	var (
		sm              *subscription.Manager
		fakeClient      client.Client
		profileTypeMeta = metav1.TypeMeta{
			Kind:       "ProfileSubscription",
			APIVersion: "weave.works/v1alpha1",
		}
		sub1       = "sub1"
		namespace1 = "namespace1"
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
		Expect(fakeClient.Create(context.TODO(), pSub1)).To(Succeed())

		sm = subscription.NewManager(fakeClient)
	})

	It("returns the subscription", func() {
		sub, err := sm.Get(namespace1, sub1)
		Expect(err).NotTo(HaveOccurred())
		Expect(sub).To(Equal(subscription.SubscriptionSummary{
			Name:      sub1,
			Namespace: namespace1,
		}))
	})

	When("the get fails", func() {
		BeforeEach(func() {
			//remove profilesv1 from scheme
			scheme := runtime.NewScheme()
			fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
			sm = subscription.NewManager(fakeClient)
		})

		It("returns an error", func() {
			_, err := sm.Get(namespace1, sub1)
			Expect(err).To(MatchError(ContainSubstring("failed to get profile subscriptions:")))
		})
	})
})
