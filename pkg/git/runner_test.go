package git_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/git/fakes"
)

var _ = Describe("runner", func() {
	When("there are git activities to perform", func() {
		It("can abstract how those activities are performed", func() {
			runner := &fakes.FakeRunner{}
			runner.RunReturns([]byte("output"), nil)
			out, err := runner.Run("cmd", "args1", "args2")
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal([]byte("output")))
		})
	})
})
