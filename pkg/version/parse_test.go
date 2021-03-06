package version_test

import (
	"github.com/Masterminds/semver/v3"
	"github.com/weaveworks/pctl/pkg/version"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParsePctlVersion", func() {
	It("handles versions with metadata", func() {
		gitVersion := "0.27.0-dev+001eeced.2020-08-27T03:03:31Z"

		v, err := version.ParsePctlVersion(gitVersion)
		semversion, _ := semver.NewVersion("0.27.0")
		Expect(err).NotTo(HaveOccurred())
		Expect(v).To(Equal(semversion))
	})
	It("handles versions without metadata", func() {
		gitVersion := "0.27.0"

		v, err := version.ParsePctlVersion(gitVersion)
		semversion, _ := semver.NewVersion("0.27.0")
		Expect(err).NotTo(HaveOccurred())
		Expect(v).To(Equal(semversion))
	})
})
