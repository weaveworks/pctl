package integration_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("pctl list", func() {
	Context("version", func() {
		It("returns information about the version of pctl", func() {
			cmd := exec.Command(binaryPath, "--version")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(session)).To(ContainSubstring("0.1.0"))
		})

		It("returns information about the version of pctl with shorthand flag", func() {
			cmd := exec.Command(binaryPath, "-v")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(session)).To(ContainSubstring("0.1.0"))
		})
	})
})
