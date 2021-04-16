package catalog_test

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/catalog/fakes"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
								Name: "test-artifact",
								Kind: profilesv1.HelmChartKind,
								Path: "test/path",
							},
						},
					},
				}, nil
			}
			objs, err := catalog.Install("http://example.catalog", "nginx", "profile", "mysub", "default", "main", gitClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeHTTPClient.DoCallCount()).To(Equal(1))
			Expect(objs).NotTo(BeEmpty())
			Expect(objs[0]).To(Equal(&v1beta1.GitRepository{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GitRepository",
					APIVersion: "source.toolkit.fluxcd.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysub-nginx-profile-main",
					Namespace: "default",
				},
				Spec: v1beta1.GitRepositorySpec{
					Reference: &v1beta1.GitRepositoryRef{
						Branch: "main",
					},
					URL: "https://github.com/weaveworks/nginx-profile",
				},
			}))
		})
	})
})
