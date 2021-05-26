package catalog_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/catalog/fakes"
	"github.com/weaveworks/pctl/pkg/subscription"
)

var _ = Describe("List", func() {
	var (
		fakeCatalogClient *fakes.FakeCatalogClient
		fakeRuntimeClient runtimeclient.Client
		profileTypeMeta   = metav1.TypeMeta{
			Kind:       "ProfileSubscription",
			APIVersion: "weave.works/v1alpha1",
		}
		sub1       = "sub1"
		sub2       = "sub2"
		sub3       = "sub3"
		namespace1 = "default"
		namespace2 = "namespace2"
		namespace3 = "namespace3"
		profile    = "weaveworks-nginx"
		profile2   = "weaveworks-nginx-2"
		profile3   = "weaveworks-nginx-3"
		cat        = "nginx-catalog"
		version    = "v0.1.0"
	)

	BeforeEach(func() {
		fakeCatalogClient = new(fakes.FakeCatalogClient)
		scheme := runtime.NewScheme()
		Expect(profilesv1.AddToScheme(scheme)).To(Succeed())
		fakeRuntimeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
		pSub1 := &profilesv1.ProfileSubscription{
			TypeMeta: profileTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      sub1,
				Namespace: namespace1,
			},
			Spec: profilesv1.ProfileSubscriptionSpec{
				ProfileURL: "https://github.com/org/repo-name",
				Version:    "weaveworks-nginx/v0.1.0",
				ProfileCatalogDescription: &profilesv1.ProfileCatalogDescription{
					Profile: profile,
					Catalog: cat,
					Version: version,
				},
			},
		}
		Expect(fakeRuntimeClient.Create(context.TODO(), pSub1)).To(Succeed())
	})

	It("lists installed profiles and checks for updates to profiles", func() {
		httpBody := []byte(`[
  {
    "name": "weaveworks-nginx",
    "description": "This installs nginx.",
    "version": "v0.1.0",
    "catalog": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "name": "weaveworks-nginx",
    "description": "This installs the next version nginx.",
    "version": "v0.1.1",
    "catalog": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  }
]
`)
		fakeCatalogClient.DoRequestReturns(httpBody, 200, nil)
		out, err := catalog.List(fakeRuntimeClient, fakeCatalogClient)
		Expect(err).NotTo(HaveOccurred())
		expected := []catalog.ProfileData{
			{
				Profile: subscription.SubscriptionSummary{
					Name:      "sub1",
					Namespace: "default",
					Profile:   "weaveworks-nginx",
					Catalog:   "nginx-catalog",
					Branch:    "-",
					Path:      "-",
					Version:   "v0.1.0",
					URL:       "https://github.com/org/repo-name",
				},
				AvailableVersionUpdates: []string{"v0.1.1"},
			},
		}
		Expect(out).To(Equal(expected))
	})
	When("there are no profiles", func() {
		It("returns an empty list with no error", func() {
			scheme := runtime.NewScheme()
			Expect(profilesv1.AddToScheme(scheme)).To(Succeed())
			fakeRuntimeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
			out, err := catalog.List(fakeRuntimeClient, fakeCatalogClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(BeEmpty())
		})
	})
	When("there are no updates for a profile", func() {
		It("returns the profile without any version updates", func() {
			httpBody := []byte(`[
  {
    "name": "weaveworks-nginx",
    "description": "This installs nginx.",
    "version": "v0.1.0",
    "catalog": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  }
]
`)
			fakeCatalogClient.DoRequestReturns(httpBody, 200, nil)
			out, err := catalog.List(fakeRuntimeClient, fakeCatalogClient)
			Expect(err).NotTo(HaveOccurred())
			expected := []catalog.ProfileData{
				{
					Profile: subscription.SubscriptionSummary{
						Name:      "sub1",
						Namespace: "default",
						Profile:   "weaveworks-nginx",
						Catalog:   "nginx-catalog",
						Branch:    "-",
						Path:      "-",
						Version:   "v0.1.0",
						URL:       "https://github.com/org/repo-name",
					},
					AvailableVersionUpdates: nil,
				},
			}
			Expect(out).To(Equal(expected))
		})
	})

	When("the profile version is invalid", func() {
		It("returns a sane error", func() {
			httpBody := []byte(`[
  {
    "name": "weaveworks-nginx",
    "description": "This installs nginx.",
    "version": "invalid",
    "catalog": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  }
]
`)
			fakeCatalogClient.DoRequestReturns(httpBody, 200, nil)
			_, err := catalog.List(fakeRuntimeClient, fakeCatalogClient)
			Expect(err).To(MatchError("failed to format profile weaveworks-nginx with version invalid into version: Malformed version: invalid"))
		})
	})

	When("the new version is invalid", func() {
		It("returns a sane error", func() {
			httpBody := []byte(`[
  {
    "name": "weaveworks-nginx",
    "description": "This installs nginx.",
    "version": "invalid",
    "catalog": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  }
]
`)
			fakeCatalogClient.DoRequestReturns(httpBody, 200, nil)
			scheme := runtime.NewScheme()
			Expect(profilesv1.AddToScheme(scheme)).To(Succeed())
			fakeRuntimeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
			pSub1 := &profilesv1.ProfileSubscription{
				TypeMeta: profileTypeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-version",
					Namespace: "default",
				},
				Spec: profilesv1.ProfileSubscriptionSpec{
					ProfileURL: "https://github.com/org/repo-name",
					Version:    "weaveworks-nginx/v0.1.0",
					ProfileCatalogDescription: &profilesv1.ProfileCatalogDescription{
						Profile: profile,
						Catalog: cat,
						Version: "invalid",
					},
				},
			}
			Expect(fakeRuntimeClient.Create(context.TODO(), pSub1)).To(Succeed())
			_, err := catalog.List(fakeRuntimeClient, fakeCatalogClient)
			Expect(err).To(MatchError("failed to format profile weaveworks-nginx with version invalid into version: Malformed version: invalid"))
		})
	})

	When("search fails to query the catalog", func() {
		It("returns a sane error", func() {
			fakeCatalogClient.DoRequestReturns(nil, 400, nil)
			out, err := catalog.List(fakeRuntimeClient, fakeCatalogClient)
			Expect(err).To(MatchError("failed to search for profile weaveworks-nginx for updates: failed to fetch profile from catalog, status code 400"))
			Expect(out).To(BeNil())
		})
	})

	When("kubernetes list fails", func() {
		It("returns a sane error", func() {
			fakeRuntimeClient = fake.NewClientBuilder().Build()
			fakeCatalogClient.DoRequestReturns(nil, 200, nil)
			_, err := catalog.List(fakeRuntimeClient, fakeCatalogClient)
			Expect(err).To(MatchError("failed to list profile subscriptions: no kind is registered for the type v1alpha1.ProfileSubscriptionList in scheme \"pkg/runtime/scheme.go:100\""))
		})
	})

	When("there are multiple results with multiple versions and multiple catalogs", func() {
		It("returns a proper result for all installed profiles", func() {
			pSub2 := &profilesv1.ProfileSubscription{
				TypeMeta: profileTypeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name:      sub2,
					Namespace: namespace2,
				},
				Spec: profilesv1.ProfileSubscriptionSpec{
					ProfileURL: "https://github.com/org/repo-name",
					Version:    "weaveworks-nginx-2/v0.1.0",
					ProfileCatalogDescription: &profilesv1.ProfileCatalogDescription{
						Profile: profile2,
						Catalog: cat,
						Version: version,
					},
				},
			}
			pSub3 := &profilesv1.ProfileSubscription{
				TypeMeta: profileTypeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name:      sub3,
					Namespace: namespace3,
				},
				Spec: profilesv1.ProfileSubscriptionSpec{
					ProfileURL: "https://github.com/org/repo-name",
					Version:    "weaveworks-nginx-3/v0.1.0",
					ProfileCatalogDescription: &profilesv1.ProfileCatalogDescription{
						Profile: profile3,
						Catalog: cat,
						Version: version,
					},
				},
			}
			Expect(fakeRuntimeClient.Create(context.TODO(), pSub2)).To(Succeed())
			Expect(fakeRuntimeClient.Create(context.TODO(), pSub3)).To(Succeed())
			httpBody := []byte(`[
  {
    "name": "weaveworks-nginx",
    "description": "This installs nginx.",
    "version": "v0.1.0",
    "catalog": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "name": "weaveworks-nginx",
    "description": "This installs the next version nginx.",
    "version": "v0.1.1",
    "catalog": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "name": "weaveworks-nginx-2",
    "description": "This installs nginx.",
    "version": "v0.1.0",
    "catalog": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "name": "weaveworks-nginx-3",
    "description": "This installs nginx.",
    "version": "v0.1.0",
    "catalog": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "name": "weaveworks-nginx-3",
    "description": "This installs nginx.",
    "version": "v0.1.5",
    "catalog": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  }
]
`)
			fakeCatalogClient.DoRequestReturns(httpBody, 200, nil)
			out, err := catalog.List(fakeRuntimeClient, fakeCatalogClient)
			Expect(err).NotTo(HaveOccurred())
			expected := []catalog.ProfileData{
				{
					Profile: subscription.SubscriptionSummary{
						Name:      "sub1",
						Namespace: "default",
						Profile:   "weaveworks-nginx",
						Catalog:   "nginx-catalog",
						Branch:    "-",
						Path:      "-",
						Version:   "v0.1.0",
						URL:       "https://github.com/org/repo-name",
					},
					AvailableVersionUpdates: []string{"v0.1.1"},
				},
				{
					Profile: subscription.SubscriptionSummary{
						Name:      "sub2",
						Namespace: "namespace2",
						Profile:   "weaveworks-nginx-2",
						Catalog:   "nginx-catalog",
						Branch:    "-",
						Path:      "-",
						Version:   "v0.1.0",
						URL:       "https://github.com/org/repo-name",
					},
					AvailableVersionUpdates: nil,
				},
				{
					Profile: subscription.SubscriptionSummary{
						Name:      "sub3",
						Namespace: "namespace3",
						Profile:   "weaveworks-nginx-3",
						Catalog:   "nginx-catalog",
						Branch:    "-",
						Path:      "-",
						Version:   "v0.1.0",
						URL:       "https://github.com/org/repo-name",
					},
					AvailableVersionUpdates: []string{"v0.1.5"},
				},
			}
			Expect(out).To(Equal(expected))
		})
	})
})
