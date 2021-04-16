package catalog_test

import (
	"bytes"
	"io/ioutil"
	"net/http"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/catalog/fakes"
)

var _ = Describe("Install", func() {
	var (
		fakeHTTPClient *fakes.FakeHTTPClient
	)

	BeforeEach(func() {
		fakeHTTPClient = new(fakes.FakeHTTPClient)
		catalog.SetHTTPClient(fakeHTTPClient)
	})

	When("there is an existing catalog and user calls install for a profile", func() {
		It("generates all artifacts and outputs installable yaml files", func() {
			httpBody := bytes.NewBufferString(`
{
	"name": "nginx-1",
	"description": "nginx 1",
	"version": "0.0.1",
	"catalog": "weaveworks (https://github.com/weaveworks/profiles)",
	"url": "https://github.com/weaveworks/nginx-profile",
	"prerequisites": ["Kubernetes 1.18+"],
	"maintainer": "WeaveWorks <gitops@weave.works>"
}
`)
			fakeHTTPClient.DoReturns(&http.Response{
				Body:       ioutil.NopCloser(httpBody),
				StatusCode: http.StatusOK,
			}, nil)

			gitClient := func(repoURL, branch string, log logr.Logger) (profilesv1.ProfileDefinition, error) {
				return profilesv1.ProfileDefinition{
					Spec: profilesv1.ProfileDefinitionSpec{
						Description: "One Artifact",
						Artifacts: []profilesv1.Artifact{
							{
								Name: "test-artifact-git-repository",
								Kind: profilesv1.HelmChartKind,
								Path: "test/path",
							},
							{
								Name: "test-artifact-helm-repository",
								Kind: profilesv1.HelmChartKind,
								Chart: &profilesv1.Chart{
									URL:     "https://org.github.io/chart",
									Name:    "nginx",
									Version: "8.8.1",
								},
							},
						},
					},
				}, nil
			}
			objs, err := catalog.Install("https://example.catalog", "nginx", "profile", "mysub", "default", "main", "", gitClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeHTTPClient.DoCallCount()).To(Equal(1))
			Expect(objs).To(HaveLen(5))
			Expect(objs[0]).To(Equal(&profilesv1.ProfileSubscription{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ProfileSubscription",
					APIVersion: "weave.works/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysub",
					Namespace: "default",
				},
				Spec: profilesv1.ProfileSubscriptionSpec{
					ProfileURL: "https://github.com/weaveworks/nginx-profile",
					Branch:     "main",
				},
			}))
			Expect(objs[1]).To(Equal(&sourcev1.GitRepository{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GitRepository",
					APIVersion: "source.toolkit.fluxcd.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysub-nginx-profile-main",
					Namespace: "default",
				},
				Spec: sourcev1.GitRepositorySpec{
					Reference: &sourcev1.GitRepositoryRef{
						Branch: "main",
					},
					URL: "https://github.com/weaveworks/nginx-profile",
				},
			}))
			Expect(objs[2]).To(Equal(&helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       "HelmRelease",
					APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysub--test-artifact-git-repository",
					Namespace: "default",
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart: "test/path",
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Name:      "mysub-nginx-profile-main",
								Namespace: "default",
								Kind:      "GitRepository",
							},
						},
					},
				},
			}))

			Expect(objs[3]).To(Equal(&helmv2.HelmRelease{
				TypeMeta: metav1.TypeMeta{
					Kind:       "HelmRelease",
					APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysub--test-artifact-helm-repository",
					Namespace: "default",
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:   "nginx",
							Version: "8.8.1",
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Name:      "mysub-nginx-profile-main-nginx",
								Namespace: "default",
								Kind:      "HelmRepository",
							},
						},
					},
				},
			}))
			Expect(objs[4]).To(Equal(&sourcev1.HelmRepository{
				TypeMeta: metav1.TypeMeta{
					Kind:       "HelmRepository",
					APIVersion: "source.toolkit.fluxcd.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysub-nginx-profile-main-nginx",
					Namespace: "default",
				},
				Spec: sourcev1.HelmRepositorySpec{
					URL: "https://org.github.io/chart",
				},
			}))
		})
	})
})
