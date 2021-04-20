package catalog_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/catalog/fakes"
)

// StringWriter is a test writer to generate output of the subscription into string
type StringWriter struct {
	output io.Writer
}

func (sw *StringWriter) Output(prof *profilesv1.ProfileSubscription) error {
	e := kjson.NewSerializerWithOptions(kjson.DefaultMetaFactory, nil, nil, kjson.SerializerOptions{Yaml: true, Strict: true})
	if err := e.Encode(prof, sw.output); err != nil {
		return err
	}
	return nil
}

var _ = Describe("Install", func() {
	var (
		fakeHTTPClient *fakes.FakeHTTPClient
	)

	BeforeEach(func() {
		fakeHTTPClient = new(fakes.FakeHTTPClient)
		catalog.SetHTTPClient(fakeHTTPClient)
	})

	When("there is an existing catalog and user calls install for a profile", func() {
		It("generates a ProfileSubscription ready to be applied to a cluster", func() {
			httpBody := bytes.NewBufferString(`
{
	"name": "nginx-1",
	"description": "nginx 1",
	"version": "0.0.1",
	"catalog": "weaveworks (https://github.com/weaveworks/profiles)",
	"url": "https://github.com/weaveworks/nginx-profile",
	"prerequisites": ["Kubernetes 1.18+"],
	"maintainer": "WeaveWorks <gitops@weave.works>"
}
`)
			fakeHTTPClient.DoReturns(&http.Response{
				Body:       ioutil.NopCloser(httpBody),
				StatusCode: http.StatusOK,
			}, nil)

			var buf bytes.Buffer
			writer := &StringWriter{
				output: &buf,
			}
			err := catalog.Install("https://example.catalog", "nginx", "profile", "mysub", "default", "main", "", writer)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeHTTPClient.DoCallCount()).To(Equal(1))
			Expect(buf).NotTo(BeNil())
			Expect(buf.String()).To(Equal(`apiVersion: weave.works/v1alpha1
kind: ProfileSubscription
metadata:
  creationTimestamp: null
  name: mysub
  namespace: default
spec:
  branch: main
  profileURL: https://github.com/weaveworks/nginx-profile
status: {}
`))
		})
		It("generates a ProfileSubscription with config map data if a config map name is defined", func() {
			httpBody := bytes.NewBufferString(`
{
	"name": "nginx-1",
	"description": "nginx 1",
	"version": "0.0.1",
	"catalog": "weaveworks (https://github.com/weaveworks/profiles)",
	"url": "https://github.com/weaveworks/nginx-profile",
	"prerequisites": ["Kubernetes 1.18+"],
	"maintainer": "WeaveWorks <gitops@weave.works>"
}
`)
			fakeHTTPClient.DoReturns(&http.Response{
				Body:       ioutil.NopCloser(httpBody),
				StatusCode: http.StatusOK,
			}, nil)

			var buf bytes.Buffer
			writer := &StringWriter{
				output: &buf,
			}
			err := catalog.Install("https://example.catalog", "nginx", "profile", "mysub", "default", "main", "config-secret", writer)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeHTTPClient.DoCallCount()).To(Equal(1))
			Expect(buf).NotTo(BeNil())
			Expect(buf.String()).To(Equal(`apiVersion: weave.works/v1alpha1
kind: ProfileSubscription
metadata:
  creationTimestamp: null
  name: mysub
  namespace: default
spec:
  branch: main
  profileURL: https://github.com/weaveworks/nginx-profile
  valuesFrom:
  - kind: ConfigMap
    name: mysub-values
    valuesKey: config-secret
status: {}
`))
		})
	})
})
