// +build release

package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/weaveworks/pctl/pkg/version"
)

var _ = Describe("generate release", func() {
	Context("release generate functions", func() {
		BeforeEach(func() {
			version.Version = "0.5.0"
			version.PreReleaseID = "dev"
		})

		It("produces a release without a pre-release id", func() {
			v, p := prepareRelease()

			Expect(v).To(Equal("0.5.0"))
			Expect(p).To(BeEmpty())
		})

		It("produces the correct release for 2 digit minor versions", func() {
			version.Version = "0.25.0"
			v, p := prepareRelease()

			Expect(v).To(Equal("0.25.0"))
			Expect(p).To(BeEmpty())
		})

		It("increases minor version for the next development iteration from a release", func() {
			version.PreReleaseID = ""

			v, p := nextDevelopmentIteration()

			Expect(v).To(Equal("0.6.0"))
			Expect(p).To(Equal("dev"))
		})

		It("increases minor version for the next development iteration from an rc", func() {
			version.PreReleaseID = "rc.1"

			v, p := nextDevelopmentIteration()

			Expect(v).To(Equal("0.6.0"))
			Expect(p).To(Equal("dev"))
		})
	})

	Context("release generate binary", func() {
		BeforeEach(func() {
			version.Version = "0.5.0"
			version.PreReleaseID = "dev"
		})
		It("produces a release version with release arguement", func() {
			cmd := exec.Command(binaryPath, "release")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(session)).To(ContainSubstring("0.5.0"))
		})
		It("produces a release version with development arguement", func() {
			cmd := exec.Command(binaryPath, "development")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(session)).To(ContainSubstring("0.6.0"))
		})
		It("produces a release version with full-version arguement", func() {
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
	})
})
