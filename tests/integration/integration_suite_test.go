package integration_test

import (
	"fmt"
	"log"
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var (
	binaryPath string
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func() {
	var err error
	binaryPath, err = gexec.Build("github.com/weaveworks/pctl/cmd/pctl")
	Expect(err).NotTo(HaveOccurred())

	http.HandleFunc("/profiles", func(w http.ResponseWriter, r *http.Request) {
		Expect(r.URL.Query().Get("name")).To(Equal("nginx"))
		fmt.Fprintf(w, `[
	{
		"name": "weaveworks-nginx",
		"description": "This installs nginx."
	}
]`)
	})

	fmt.Printf("Starting server at port 8080\n")

	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal(err)
		}
	}()

})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
