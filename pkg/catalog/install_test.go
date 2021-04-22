package catalog_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/catalog/fakes"
	gitfakes "github.com/weaveworks/pctl/pkg/git/fakes"
	"github.com/weaveworks/pctl/pkg/writer"
)

var _ = Describe("Install", func() {
	var (
		fakeHTTPClient *fakes.FakeHTTPClient
		fakeGit        *gitfakes.FakeGit
		fakeScm        *gitfakes.FakeSCMClient
	)

	BeforeEach(func() {
		fakeHTTPClient = new(fakes.FakeHTTPClient)
		fakeGit = new(gitfakes.FakeGit)
		fakeScm = new(gitfakes.FakeSCMClient)
		catalog.SetHTTPClient(fakeHTTPClient)
	})

	When("there is an existing catalog and user calls install for a profile", func() {
		It("generates a ProfileSubscription ready to be applied to a cluster", func() {
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

			var buf bytes.Buffer
			writer := &writer.StringWriter{
				Out: &buf,
			}
			cfg := catalog.InstallConfig{
				Branch:      "main",
				CatalogName: "nginx",
				CatalogURL:  "https://example.catalog",
				Namespace:   "default",
				ProfileName: "profile",
				SubName:     "mysub",
				Writer:      writer,
			}
			err := catalog.Install(cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeHTTPClient.DoCallCount()).To(Equal(1))
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

			var buf bytes.Buffer
			writer := &writer.StringWriter{
				Out: &buf,
			}
			cfg := catalog.InstallConfig{
				Branch:      "main",
				CatalogName: "nginx",
				CatalogURL:  "https://example.catalog",
				ConfigMap:   "config-secret",
				Namespace:   "default",
				ProfileName: "profile",
				SubName:     "mysub",
				Writer:      writer,
			}
			err := catalog.Install(cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeHTTPClient.DoCallCount()).To(Equal(1))
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
		It("returns an error in case the profile does not exist", func() {
			httpBody := bytes.NewBufferString(`{}`)
			fakeHTTPClient.DoReturns(&http.Response{
				Body:       ioutil.NopCloser(httpBody),
				StatusCode: http.StatusNotFound,
			}, nil)

			var buf bytes.Buffer
			writer := &writer.StringWriter{
				Out: &buf,
			}
			cfg := catalog.InstallConfig{
				Branch:      "main",
				CatalogName: "nginx",
				CatalogURL:  "https://example.catalog",
				ConfigMap:   "config-secret",
				Namespace:   "default",
				ProfileName: "profile",
				SubName:     "mysub",
				Writer:      writer,
			}
			err := catalog.Install(cfg)
			Expect(err).To(MatchError("unable to find profile `profile` in catalog `nginx`"))
		})
		It("returns an error in case the call is non-200", func() {
			httpBody := bytes.NewBufferString(`{}`)
			fakeHTTPClient.DoReturns(&http.Response{
				Body:       ioutil.NopCloser(httpBody),
				StatusCode: http.StatusTeapot,
			}, nil)

			err := catalog.Install(catalog.InstallConfig{})
			Expect(err).To(MatchError("failed to fetch profile: status code 418"))
		})
		It("returns an error in the url is invalid", func() {
			httpBody := bytes.NewBufferString(`{}`)
			fakeHTTPClient.DoReturns(&http.Response{
				Body:       ioutil.NopCloser(httpBody),
				StatusCode: http.StatusOK,
			}, nil)

			err := catalog.Install(catalog.InstallConfig{CatalogURL: "invalid_1234%^"})
			Expect(err).To(MatchError(`failed to parse url "invalid_1234%^": parse "invalid_1234%^": invalid URL escape "%^"`))
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
