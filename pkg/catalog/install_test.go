package catalog_test

import (
	"bytes"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/catalog/fakes"
	gitfakes "github.com/weaveworks/pctl/pkg/git/fakes"
	"github.com/weaveworks/pctl/pkg/writer"
)

var _ = Describe("Install", func() {
	var (
		fakeCatalogClient *fakes.FakeCatalogClient
		fakeGit           *gitfakes.FakeGit
		fakeScm           *gitfakes.FakeSCMClient
	)

	BeforeEach(func() {
		fakeCatalogClient = new(fakes.FakeCatalogClient)
		fakeGit = new(gitfakes.FakeGit)
		fakeScm = new(gitfakes.FakeSCMClient)
	})

	When("there is an existing catalog and user calls install for a profile", func() {
		It("generates a ProfileSubscription ready to be applied to a cluster", func() {
			httpBody := []byte(`
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
			fakeCatalogClient.DoRequestReturns(httpBody, 200, nil)

			var buf bytes.Buffer
			writer := &writer.StringWriter{
				Out: &buf,
			}
			cfg := catalog.InstallConfig{
				Branch:        "main",
				CatalogName:   "nginx",
				CatalogClient: fakeCatalogClient,
				Namespace:     "default",
				ProfileName:   "profile",
				SubName:       "mysub",
				Writer:        writer,
			}
			err := catalog.Install(cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(buf).NotTo(BeNil())
			Expect(buf.String()).To(Equal(`apiVersion: weave.works/v1alpha1
kind: ProfileSubscription
metadata:
  creationTimestamp: null
  name: mysub
  namespace: default
spec:
  branch: main
  profileURL: https://github.com/weaveworks/nginx-profile
status: {}
`))
		})

		It("generates a ProfileSubscription with config map data if a config map name is defined", func() {
			httpBody := []byte(`
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
			fakeCatalogClient.DoRequestReturns(httpBody, 200, nil)

			var buf bytes.Buffer
			writer := &writer.StringWriter{
				Out: &buf,
			}
			cfg := catalog.InstallConfig{
				Branch:        "main",
				CatalogName:   "nginx",
				CatalogClient: fakeCatalogClient,
				ConfigMap:     "config-secret",
				Namespace:     "default",
				ProfileName:   "profile",
				SubName:       "mysub",
				Writer:        writer,
			}
			err := catalog.Install(cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(buf).NotTo(BeNil())
			Expect(buf.String()).To(Equal(`apiVersion: weave.works/v1alpha1
kind: ProfileSubscription
metadata:
  creationTimestamp: null
  name: mysub
  namespace: default
spec:
  branch: main
  profileURL: https://github.com/weaveworks/nginx-profile
  valuesFrom:
  - kind: ConfigMap
    name: mysub-values
    valuesKey: config-secret
status: {}
`))
		})

		It("returns an error when getting the profile fails", func() {
			fakeCatalogClient.DoRequestReturns([]byte(""), 0, fmt.Errorf("foo"))

			var buf bytes.Buffer
			writer := &writer.StringWriter{
				Out: &buf,
			}
			cfg := catalog.InstallConfig{
				Branch:        "main",
				CatalogName:   "nginx",
				CatalogClient: fakeCatalogClient,
				ConfigMap:     "config-secret",
				Namespace:     "default",
				ProfileName:   "profile",
				SubName:       "mysub",
				Writer:        writer,
			}
			err := catalog.Install(cfg)
			Expect(err).To(MatchError(ContainSubstring("failed to get profile \"profile\" in catalog \"nginx\":")))
		})
	})

	When("create-pr is set to true", func() {
		It("can create a PR if the generated values result in changes", func() {
			err := catalog.CreatePullRequest(fakeScm, fakeGit)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeGit.CreateBranchCallCount()).To(Equal(1))
			Expect(fakeGit.AddCallCount()).To(Equal(1))
			Expect(fakeGit.CommitCallCount()).To(Equal(1))
			Expect(fakeGit.PushCallCount()).To(Equal(1))
			Expect(fakeScm.CreatePullRequestCallCount()).To(Equal(1))
		})
	})
	When("create-pr is set to true but something goes wrong", func() {
		It("handles create branch errors", func() {
			fakeGit.CreateBranchReturns(errors.New("nope"))
			err := catalog.CreatePullRequest(fakeScm, fakeGit)
			Expect(err).To(MatchError("failed to create branch: nope"))
		})
		It("handles add errors", func() {
			fakeGit.AddReturns(errors.New("nope"))
			err := catalog.CreatePullRequest(fakeScm, fakeGit)
			Expect(err).To(MatchError("failed to add changes: nope"))
		})
		It("handles commit errors", func() {
			fakeGit.CommitReturns(errors.New("nope"))
			err := catalog.CreatePullRequest(fakeScm, fakeGit)
			Expect(err).To(MatchError("failed to commit changes: nope"))
		})
		It("handles push errors", func() {
			fakeGit.PushReturns(errors.New("nope"))
			err := catalog.CreatePullRequest(fakeScm, fakeGit)
			Expect(err).To(MatchError("failed to push changes: nope"))
		})
		It("handles create pull request errors", func() {
			fakeScm.CreatePullRequestReturns(errors.New("nope"))
			err := catalog.CreatePullRequest(fakeScm, fakeGit)
			Expect(err).To(MatchError("failed to create pull request: nope"))
		})
	})
})
