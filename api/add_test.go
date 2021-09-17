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
			Expect(s).To(Equal("https://github.com/weaveworks/nginx-profile"))
			Expect(s2).To(Equal("0.0.1"))
			content, err := ioutil.ReadFile(filepath.Join(".", "testdata", "profile.yaml"))
			Expect(err).ToNot(HaveOccurred())
			err = ioutil.WriteFile(filepath.Join(s3, "profile.yaml"), content, 0644)
			Expect(err).ToNot(HaveOccurred())
			return nil
		}
		err := api.AddProfile(api.AddProfileOpts{
			Branch:                 "main",
			SubName:                "sub-name",
			Namespace:              "namespace",
			ConfigMap:              "configMap",
			InstallationDirectory:  "test-dir",
			GitRepositoryName:      "git-repo-name",
			GitRepositoryNamespace: "git-repo-namespace",
			ProfileName:            "weave-nginx",
			CatalogName:            "catalog",
			CatalogClient:          fakeCatalogClient,
			GitClient:              fakeGit,
		})

		Expect(err).NotTo(HaveOccurred())
	})
	When("a URL is provided", func() {
		It("will use that to install a profile", func() {
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
				Expect(s).To(Equal("https://github.com/org/repo"))
				Expect(s2).To(Equal("main"))
				content, err := ioutil.ReadFile(filepath.Join(".", "testdata", "profile.yaml"))
				Expect(err).ToNot(HaveOccurred())
				err = ioutil.WriteFile(filepath.Join(s3, "profile.yaml"), content, 0644)
				Expect(err).ToNot(HaveOccurred())
				return nil
			}
			err := api.AddProfile(api.AddProfileOpts{
				Branch:                 "main",
				SubName:                "sub-name",
				Namespace:              "namespace",
				ConfigMap:              "configMap",
				InstallationDirectory:  "test-dir",
				GitRepositoryNamespace: "git-repo-namespace",
				GitRepositoryName:      "git-repo-name",
				URL:                    "https://github.com/org/repo",
				CatalogClient:          fakeCatalogClient,
				GitClient:              fakeGit,
			})

			Expect(err).NotTo(HaveOccurred())
		})
		When("invalid profile path is provided", func() {
			It("returns an error", func() {
				err := api.AddProfile(api.AddProfileOpts{
					Branch:                 "main",
					SubName:                "sub-name",
					Namespace:              "namespace",
					ConfigMap:              "configMap",
					InstallationDirectory:  "test-dir",
					GitRepositoryName:      "git-repo-name",
					GitRepositoryNamespace: "git-repo-namespace",
					CatalogName:            "catalog",
					CatalogClient:          fakeCatalogClient,
					GitClient:              fakeGit,
				})

				Expect(err).To(MatchError("both catalog name and profile name must be provided"))
			})
		})
		When("path and url are provided", func() {
			It("uses that", func() {
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
					Expect(s).To(Equal("https://github.com/org/repo"))
					Expect(s2).To(Equal("main"))
					content, err := ioutil.ReadFile(filepath.Join(".", "testdata", "profile.yaml"))
					Expect(err).ToNot(HaveOccurred())
					_ = os.MkdirAll(filepath.Join(s3, "path"), 0700)
					err = ioutil.WriteFile(filepath.Join(s3, "path", "profile.yaml"), content, 0644)
					Expect(err).ToNot(HaveOccurred())
					return nil
				}
				err := api.AddProfile(api.AddProfileOpts{
					Branch:                 "main",
					SubName:                "sub-name",
					Namespace:              "namespace",
					ConfigMap:              "configMap",
					InstallationDirectory:  "test-dir",
					GitRepositoryName:      "git-repo-name",
					GitRepositoryNamespace: "git-repo-namespace",
					URL:                    "https://github.com/org/repo",
					Path:                   "path",
					CatalogClient:          fakeCatalogClient,
					GitClient:              fakeGit,
				})

				Expect(err).NotTo(HaveOccurred())
			})
		})
		When("profile path includes a version", func() {
			It("uses that to query the profile manager", func() {
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
					Expect(s).To(Equal("https://github.com/weaveworks/nginx-profile"))
					Expect(s2).To(Equal("0.0.1"))
					content, err := ioutil.ReadFile(filepath.Join(".", "testdata", "profile.yaml"))
					Expect(err).ToNot(HaveOccurred())
					err = ioutil.WriteFile(filepath.Join(s3, "profile.yaml"), content, 0644)
					Expect(err).ToNot(HaveOccurred())
					return nil
				}
				err := api.AddProfile(api.AddProfileOpts{
					Branch:                 "main",
					SubName:                "sub-name",
					Namespace:              "namespace",
					ConfigMap:              "configMap",
					InstallationDirectory:  "test-dir",
					GitRepositoryNamespace: "git-repo-namespace",
					GitRepositoryName:      "git-repo-name",
					ProfileName:            "profile",
					CatalogName:            "catalog",
					Version:                "version",
					CatalogClient:          fakeCatalogClient,
					GitClient:              fakeGit,
				})

				Expect(err).NotTo(HaveOccurred())
				arg, _ := fakeCatalogClient.DoRequestArgsForCall(0)
				Expect(arg).To(Equal("/profiles/catalog/profile/version"))
			})
		})
		When("profile, catalog and url cannot be used together", func() {
			It("returns an error", func() {
				err := api.AddProfile(api.AddProfileOpts{
					Branch:                 "main",
					SubName:                "sub-name",
					Namespace:              "namespace",
					ConfigMap:              "configMap",
					InstallationDirectory:  "test-dir",
					GitRepositoryName:      "git-repo-name",
					GitRepositoryNamespace: "git-repo-namespace",
					CatalogName:            "catalog",
					ProfileName:            "profile",
					URL:                    "url",
					CatalogClient:          fakeCatalogClient,
					GitClient:              fakeGit,
				})

				Expect(err).To(MatchError("please provide either url or profile name with catalog name"))
			})
		})
		When("no profile name or catalog name or url is provided", func() {
			It("return an error", func() {
				err := api.AddProfile(api.AddProfileOpts{
					Branch:                 "main",
					SubName:                "sub-name",
					Namespace:              "namespace",
					ConfigMap:              "configMap",
					InstallationDirectory:  "test-dir",
					GitRepositoryName:      "git-repo-name",
					GitRepositoryNamespace: "git-repo-namespace",
					CatalogClient:          fakeCatalogClient,
					GitClient:              fakeGit,
				})

				Expect(err).To(MatchError("please provide either url or profile name with catalog name"))
			})
		})
	})
})
