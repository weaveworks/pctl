package formatter_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/formatter"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

var _ = Describe("JsonFormater", func() {
	It("formats output as json", func() {
		jsonFormatter := formatter.NewJSONFormatter()
		dataFunc := func() interface{} {
			return profilesv1.ProfileDescription{Name: "foo", Catalog: "bar"}
		}
		out, err := jsonFormatter.Format(dataFunc)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal(`{
  "name": "foo",
  "catalog": "bar"
}`))
	})
})
