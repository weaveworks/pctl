package git_test

import (
	"github.com/jenkins-x/go-scm/scm/factory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/weaveworks/pctl/pkg/git"
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
	})
})
