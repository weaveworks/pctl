package catalog_test

import (
	"bytes"
	"io/ioutil"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/catalog/fakes"
)

var _ = Describe("Install", func() {
	var (
		fakeHTTPClient *fakes.FakeHTTPClient
	)

	BeforeEach(func() {
		fakeHTTPClient = new(fakes.FakeHTTPClient)
		catalog.SetHTTPClient(fakeHTTPClient)
	})

	When("there is an existing catalog and user calls install for a profile", func() {
		It("generates all artifacts and outputs installable yaml files", func() {
			httpBody := bytes.NewBufferString(`
[
    {
      "name": "nginx-1",
      "description": "nginx 1"
    },
    {
      "name": "nginx-2",
      "description": "nginx 2"
    }
]
		  `)
			fakeHTTPClient.DoReturns(&http.Response{
				Body:       ioutil.NopCloser(httpBody),
				StatusCode: http.StatusOK,
			}, nil)

			err := catalog.Install("http://example.catalog", "nginx", "profile", "mysub", "default", "main")
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeHTTPClient.DoCallCount()).To(Equal(1))
		})
	})
})
