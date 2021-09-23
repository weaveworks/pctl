package git_test

import (
	"os"

	"github.com/jenkins-x/go-scm/scm/factory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/weaveworks/kivo-cli/pkg/git"
)

var _ = Describe("scm", func() {
	When("we are trying to create a pull request", func() {
		It("can use an scm client to talk to the platform", func() {
			fakeScm, err := factory.NewClient("fake", "", "")
			Expect(err).NotTo(HaveOccurred())
			client, err := git.NewClient(git.SCMConfig{
				Branch: "",
				Base:   "",
				Repo:   "",
				Client: fakeScm,
			})
			Expect(err).NotTo(HaveOccurred())
			err = client.CreatePullRequest()
			Expect(err).NotTo(HaveOccurred())
		})

		It("fails if the github token is not set", func() {
			Expect(os.Setenv("GITHUB_TOKEN", "")).To(Succeed())
			_, err := git.NewClient(git.SCMConfig{
				Branch: "",
				Base:   "",
				Repo:   "",
			})
			Expect(err).To(MatchError("failed to create scm client: GITHUB_TOKEN not set"))
		})

		It("fails if the scm client can't contact the provider", func() {
			fakeScm, err := factory.NewClient("github", "https://invalid.url.com.here", "")
			Expect(err).NotTo(HaveOccurred())
			client, err := git.NewClient(git.SCMConfig{
				Branch: "",
				Base:   "",
				Repo:   "",
				Client: fakeScm,
			})
			Expect(err).NotTo(HaveOccurred())
			err = client.CreatePullRequest()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("error while creating pr: Post \"https://invalid.url.com.here/api/v3/repos//pulls\": dial tcp: lookup invalid.url.com.here: no such host"))
		})
		It("fails if token is invalid", func() {
			fakeScm, err := factory.NewClient("github", "https://api.github.com", "invalid")
			Expect(err).NotTo(HaveOccurred())
			client, err := git.NewClient(git.SCMConfig{
				Branch: "test01",
				Base:   "main",
				Repo:   "weaveworks/pctl-test-repo",
				Client: fakeScm,
			})
			Expect(err).NotTo(HaveOccurred())
			err = client.CreatePullRequest()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("error while creating pr: Unauthorized"))
		})
		It("fails if token is not provided", func() {
			fakeScm, err := factory.NewClient("github", "https://api.github.com", "")
			Expect(err).NotTo(HaveOccurred())
			client, err := git.NewClient(git.SCMConfig{
				Branch: "test01",
				Base:   "main",
				Repo:   "weaveworks/pctl-test-repo",
				Client: fakeScm,
			})
			Expect(err).NotTo(HaveOccurred())
			err = client.CreatePullRequest()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("error while creating pr: Not Found"))
		})
	})
})
