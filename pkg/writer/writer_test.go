package writer_test

import (
	"bytes"
	"io/ioutil"
	"path/filepath"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/writer"
)

var _ = Describe("Writer", func() {
	var prof *profilesv1.ProfileSubscription
	BeforeEach(func() {
		prof = &profilesv1.ProfileSubscription{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ProfileSubscription",
				APIVersion: "weave.works/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "writer-test",
				Namespace: "default",
			},
			Spec: profilesv1.ProfileSubscriptionSpec{
				ProfileURL: "https://github.com/weaveworks/nginx-profile",
				Branch:     "main",
				ValuesFrom: []helmv2.ValuesReference{
					{
						Kind:      "ConfigMap",
						Name:      "values-from",
						ValuesKey: "test-name",
					},
				},
			},
		}
	})

	When("the writer is a string writer", func() {
		It("will drop the content into an io writer", func() {
			var buf bytes.Buffer
			writer := writer.StringWriter{Out: &buf}
			err := writer.Output(prof)
			Expect(err).NotTo(HaveOccurred())
			Expect(buf.String()).To(Equal(`apiVersion: weave.works/v1alpha1
kind: ProfileSubscription
metadata:
  creationTimestamp: null
  name: writer-test
  namespace: default
spec:
  branch: main
  profileURL: https://github.com/weaveworks/nginx-profile
  valuesFrom:
  - kind: ConfigMap
    name: values-from
    valuesKey: test-name
status: {}
`))
		})
	})

	When("the writer is a file writer", func() {
		It("will drop the content into a given file", func() {
			temp, err := ioutil.TempDir("", "writer_test_filewriter_01")
			Expect(err).ToNot(HaveOccurred())
			filename := filepath.Join(temp, "profile_subscription.yaml")
			writer := writer.FileWriter{Filename: filename}
			err = writer.Output(prof)
			Expect(err).NotTo(HaveOccurred())
			content, err := ioutil.ReadFile(filename)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(Equal(`apiVersion: weave.works/v1alpha1
kind: ProfileSubscription
metadata:
  creationTimestamp: null
  name: writer-test
  namespace: default
spec:
  branch: main
  profileURL: https://github.com/weaveworks/nginx-profile
  valuesFrom:
  - kind: ConfigMap
    name: values-from
    valuesKey: test-name
status: {}
`))
		})
		It("will return an error if the file cannot be created", func() {
			filename := filepath.Join("./doesnotexistfolder", "profile_subscription.yaml")
			writer := writer.FileWriter{Filename: filename}
			err := writer.Output(prof)
			Expect(err).To(HaveOccurred())
		})
	})
})
