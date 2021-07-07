package formatter_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/formatter"
)

var _ = Describe("JsonFormater", func() {
	It("formats output as json", func() {
		jsonFormatter := formatter.NewJSONFormatter()
		dataFunc := func() interface{} {
			return profilesv1.ProfileCatalogEntry{Name: "foo", CatalogSource: "bar"}
		}
		out, err := jsonFormatter.Format(dataFunc)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal(`{
  "catalogSource": "bar",
  "name": "foo"
}`))
	})
})
