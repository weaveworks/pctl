package profile

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("sanitise", func() {
	DescribeTable("invalid and valid branches", func(branch string, outcome string) {
		result := SanitiseBranchName(branch)
		Expect(result).To(Equal(outcome))
	},
		Entry("underscore", "this_branch", "this-branch"),
		Entry("invalid characters", ";lk;f';branch", "lkfbranch"),
		Entry("valid name should not change", "valid-branch", "valid-branch"),
		Entry("single word", "main", "main"),
	)
})
