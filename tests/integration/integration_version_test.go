package integration_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/version"
)

var _ = Describe("pctl list", func() {
	Context("version", func() {
		It("returns information about the version of pctl", func() {
			cmd := exec.Command(binaryPath, "--version")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(session)).To(ContainSubstring(version.Version))
		})

		It("returns information about the version of pctl with shorthand flag", func() {
			cmd := exec.Command(binaryPath, "-v")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(session)).To(ContainSubstring(version.Version))
		})
	})
})
