package installation_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/kivo-cli/pkg/installation"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("List", func() {
	var (
		sm              *installation.Manager
		fakeClient      client.Client
		profileTypeMeta = metav1.TypeMeta{
			Kind:       "ProfileInstallation",
			APIVersion: "weave.works/v1alpha1",
		}
		sub1       = "sub1"
		sub2       = "sub2"
		sub3       = "sub3"
		namespace1 = "namespace1"
		namespace2 = "namespace2"
		namespace3 = "namespace3"
		profile    = "foo"
		catalog    = "bar"
		version    = "v0.1.0"
	)

	BeforeEach(func() {
		scheme := runtime.NewScheme()
		Expect(profilesv1.AddToScheme(scheme)).To(Succeed())
		fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
		pSub1 := &profilesv1.ProfileInstallation{
			TypeMeta: profileTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      sub1,
				Namespace: namespace1,
			},
			Spec: profilesv1.ProfileInstallationSpec{
				Source: &profilesv1.Source{
					URL: "https://github.com/org/repo-name",
					Tag: "foo/v0.1.0",
				},
				Catalog: &profilesv1.Catalog{
					Profile: profile,
					Catalog: catalog,
					Version: version,
				},
			},
		}
		pSub2 := &profilesv1.ProfileInstallation{
			TypeMeta: profileTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      sub2,
				Namespace: namespace2,
			},
			Spec: profilesv1.ProfileInstallationSpec{
				Source: &profilesv1.Source{
					URL:    "https://github.com/org/repo-name",
					Branch: "main",
				},
			},
		}
		pSub3 := &profilesv1.ProfileInstallation{
			TypeMeta: profileTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      sub3,
				Namespace: namespace3,
			},
			Spec: profilesv1.ProfileInstallationSpec{
				Source: &profilesv1.Source{
					URL:    "https://github.com/org/repo-name",
					Branch: "main",
					Path:   "path",
				},
			},
		}
		Expect(fakeClient.Create(context.TODO(), pSub1)).To(Succeed())
		Expect(fakeClient.Create(context.TODO(), pSub2)).To(Succeed())
		Expect(fakeClient.Create(context.TODO(), pSub3)).To(Succeed())

		sm = installation.NewManager(fakeClient)
	})

	It("returns a list of profiles deployed in the cluster", func() {
		subs, err := sm.List()
		Expect(err).NotTo(HaveOccurred())
		Expect(subs).To(ConsistOf(
			installation.Summary{
				Name:      sub1,
				Namespace: namespace1,
				Version:   version,
				Profile:   profile,
				Catalog:   catalog,
				Branch:    "-",
				Path:      "-",
				URL:       "https://github.com/org/repo-name",
			},
			installation.Summary{
				Name:      sub2,
				Namespace: namespace2,
				Version:   "-",
				Profile:   "-",
				Catalog:   "-",
				Branch:    "main",
				Path:      "-",
				URL:       "https://github.com/org/repo-name",
			},
			installation.Summary{
				Name:      sub3,
				Namespace: namespace3,
				Version:   "-",
				Profile:   "-",
				Catalog:   "-",
				Branch:    "main",
				Path:      "path",
				URL:       "https://github.com/org/repo-name",
			},
		))
	})

	When("the list fails", func() {
		BeforeEach(func() {
			//remove profilesv1 from scheme
			scheme := runtime.NewScheme()
			fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
			sm = installation.NewManager(fakeClient)
		})

		It("returns an error", func() {
			_, err := sm.List()
			Expect(err).To(MatchError(ContainSubstring("failed to list profile installations:")))
		})
	})
})
