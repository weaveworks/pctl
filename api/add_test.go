package api_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/weaveworks/pctl/api"
	catalogfake "github.com/weaveworks/pctl/pkg/catalog/fakes"
	gitfake "github.com/weaveworks/pctl/pkg/git/fakes"
)

var _ = Describe("AddProfile", func() {
	var (
		fakeGit           *gitfake.FakeGit
		fakeCatalogClient *catalogfake.FakeCatalogClient
	)
	BeforeEach(func() {
		fakeGit = &gitfake.FakeGit{}
		fakeCatalogClient = &catalogfake.FakeCatalogClient{}
	})
	It("can retrieve profiles", func() {
		httpBody := []byte(`{"item":
{
	"name": "nginx-1",
	"description": "nginx 1",
	"tag": "0.0.1",
	"catalogSource": "weaveworks (https://github.com/weaveworks/profiles)",
	"url": "https://github.com/weaveworks/nginx-profile",
	"prerequisites": ["Kubernetes 1.18+"],
	"maintainer": "WeaveWorks <gitops@weave.works>"
}}
		  `)
		fakeCatalogClient.DoRequestReturns(httpBody, 200, nil)
		fakeGit.CloneStub = func(s string, s2 string, s3 string) error {
			content, err := ioutil.ReadFile(filepath.Join(".", "testdata", "profile.yaml"))
			Expect(err).ToNot(HaveOccurred())
			profile, err := os.Create(filepath.Join(s3, "profile.yaml"))
			Expect(err).ToNot(HaveOccurred())
			_, err = profile.Write(content)
			Expect(err).ToNot(HaveOccurred())
			return nil
		}
		installationPath, err := api.AddProfile(api.AddProfileOpts{
			Branch:        "main",
			SubName:       "sub-name",
			Namespace:     "namespace",
			ConfigMap:     "configMap",
			Dir:           "test-dir",
			GitRepository: "git-repo-namespace/git-repo-name",
			ProfilePath:   "profile/weave-nginx",
			CatalogClient: fakeCatalogClient,
			GitClient:     fakeGit,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(installationPath).To(Equal(filepath.Join("test-dir", "sub-name")))
	})
})
