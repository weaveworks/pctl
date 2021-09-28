package integration_test

import (
	"net/http"
	"net/http/httptest"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Kivo", func() {
	Context("install", func() {
		When("dry-run is provided", func() {
			It("displays the to be applied content", func() {
				cmd := exec.Command(binaryPath, "install", "--dry-run")
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				Expect(string(output)).To(ContainSubstring("kind: List"))
			})
		})
		When("baseurl is provided", func() {
			It("will use that to fetch releases", func() {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusTeapot)
				}))
				cmd := exec.Command(binaryPath, "install", "--baseurl="+server.URL)
				output, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(output)).To(ContainSubstring("status: 418 I'm a teapot"))
			})
		})
		When("version is provided", func() {
			It("will fetch that specific version", func() {
				// use dry-run here so we don't overwrite the created test cluster resources with old version.
				cmd := exec.Command(binaryPath, "install", "--version=v0.0.1", "--dry-run")
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				Expect(string(output)).To(ContainSubstring("kind: List"))
			})
		})
		When("the provided version is missing", func() {
			It("will put out an understandable error message", func() {
				cmd := exec.Command(binaryPath, "install", "--version=vnope")
				output, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(output)).To(ContainSubstring("status: 404 Not Found"))
			})
		})
	})
})
