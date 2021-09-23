package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/kivo-cli/pkg/version"
)

var _ = Describe("kivo list", func() {
	Context("version", func() {
		It("returns information about the version of kivo", func() {
			Expect(kivoWithRawOutput("--version")).To(ContainSubstring(version.Version))
		})

		It("returns information about the version of kivo with shorthand flag", func() {
			Expect(kivoWithRawOutput("-v")).To(ContainSubstring(version.Version))
		})
	})
})
