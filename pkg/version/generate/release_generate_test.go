// +build release

package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/weaveworks/pctl/pkg/version"
)

var _ = Describe("generate release", func() {
	Context("when release_generate binary is called with various arguments", func() {
		BeforeEach(func() {
			version.Version = "0.5.0"
			version.PreReleaseID = "dev"
		})
		It("produces a release version with release argument", func() {
			cmd := exec.Command(binaryPath, "release")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(session)).To(ContainSubstring("0.5.0"))
		})
		It("produces a release version with development argument", func() {
			cmd := exec.Command(binaryPath, "development")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(session)).To(ContainSubstring("0.6.0"))
		})
		It("produces a release version with full-version argument", func() {
			cmd := exec.Command(binaryPath, "full-version")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(session)).To(ContainSubstring("0.5.0"))
		})
		It("produces a error when a valid agrument is not provided", func() {
			cmd := exec.Command(binaryPath, "blah")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(session)).To(ContainSubstring("unknown option blah. Expected 'release','development','full-version'"))
		})
		It("produces a error when no agrument is not provided", func() {
			cmd := exec.Command(binaryPath)
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(session)).To(ContainSubstring("usage: generate <release|development|full-version>"))
		})
		It("produces the correct release for 2 digit minor versions", func() {
			version.Version = "0.25.0"
			cmd := exec.Command(binaryPath, "release")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(session)).To(ContainSubstring("0.25.0"))
		})
		It("increases minor version for the next development iteration from a release", func() {
			version.PreReleaseID = ""
			cmd := exec.Command(binaryPath, "development")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(session)).To(ContainSubstring("0.6.0"))
		})

		It("increases minor version for the next development iteration from an rc", func() {
			version.PreReleaseID = "rc.1"
			cmd := exec.Command(binaryPath, "development")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(session)).To(ContainSubstring("0.6.0"))
		})
	})
})
