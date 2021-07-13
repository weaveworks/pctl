package version_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/weaveworks/pctl/pkg/version"
)

var _ = Describe("release tests", func() {
	BeforeEach(func() {
		version.Version = "0.5.0"
		version.PreReleaseID = ""
		version.GitCommit = ""
		version.BuildDate = ""
	})

	It("ignores pre-release and build metadata for releases", func() {
		v := version.GetVersion()
		info := version.GetVersionInfo()

		Expect(v).To(Equal("0.5.0"))
		Expect(info).To(Equal(version.Info{
			Version:      "0.5.0",
			PreReleaseID: "",
			Metadata:     version.BuildMetadata{},
		}))
	})

	It("produces a dev version with build metadata", func() {
		version.PreReleaseID = "dev"
		version.GitCommit = "abc1234"
		version.BuildDate = "2020-01-15T14:03:46Z"

		v := version.GetVersion()
		info := version.GetVersionInfo()

		Expect(v).To(Equal("0.5.0-dev+abc1234.2020-01-15T14:03:46Z"))
		Expect(info).To(Equal(version.Info{
			Version:      "0.5.0",
			PreReleaseID: "dev",
			Metadata: version.BuildMetadata{
				GitCommit: "abc1234",
				BuildDate: "2020-01-15T14:03:46Z",
			},
		}))
	})

	It("skips build metadata when not present", func() {
		version.PreReleaseID = "dev"

		v := version.GetVersion()
		info := version.GetVersionInfo()

		Expect(v).To(Equal("0.5.0-dev"))
		Expect(info).To(Equal(version.Info{
			Version:      "0.5.0",
			PreReleaseID: "dev",
			Metadata:     version.BuildMetadata{},
		}))
	})

})
