package integration_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("PCTL", func() {
	Context("search", func() {
		It("returns the matching profiles", func() {
			exampleCatalog := "http://localhost:8080"
			cmd := exec.Command(binaryPath, "search", "--catalog-url", exampleCatalog, "nginx")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(string(session.Out.Contents())).To(ContainSubstring("weaveworks-nginx: This installs nginx"))
		})

		When("catalog-url is not provided", func() {
			It("returns a useful error", func() {
				cmd := exec.Command(binaryPath, "search", "nginx")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Expect(string(session.Err.Contents())).To(ContainSubstring("--catalog-url must be provided"))
			})
		})
	})
})
