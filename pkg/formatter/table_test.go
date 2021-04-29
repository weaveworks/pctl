package formatter_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/formatter"
)

var _ = Describe("Table", func() {
	var tableFormatter formatter.Formatter
	BeforeEach(func() {
		tableFormatter = formatter.NewTableFormatter()
	})

	It("formats output in a table", func() {
		contFunc := func() interface{} {
			return formatter.TableContents{
				Headers: []string{"col1", "col2"},
				Data:    [][]string{{"dat1", "dat2"}},
			}
		}
		out, err := tableFormatter.Format(contFunc)
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal("COL1\tCOL2 \ndat1\tdat2\t\n"))
	})

	When("the wrong type is returned in the getter func", func() {
		It("returns an error", func() {
			contFunc := func() interface{} {
				return "not a table contents obj"
			}
			_, err := tableFormatter.Format(contFunc)
			Expect(err).To(MatchError("func returned wrong type for table formatter. wanted formatter.TableContents"))
		})
	})
})
