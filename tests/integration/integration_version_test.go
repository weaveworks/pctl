package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/version"
)

var _ = Describe("pctl list", func() {
	Context("version", func() {
		It("returns information about the version of pctl", func() {
			Expect(pctlWithRawOutput("--version")).To(ContainSubstring(version.Version))
		})

		It("returns information about the version of pctl with shorthand flag", func() {
			Expect(pctlWithRawOutput("-v")).To(ContainSubstring(version.Version))
		})
	})
})
