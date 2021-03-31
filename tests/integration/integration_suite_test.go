package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var (
	binaryPath string
	server     *http.Server
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func() {
	var err error
	binaryPath, err = gexec.Build("github.com/weaveworks/pctl/cmd/pctl")
	Expect(err).NotTo(HaveOccurred())

	mux := http.NewServeMux()
	mux.Handle("/profiles", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			Expect(r.URL.Query().Get("name")).To(Equal("nginx"))
			fmt.Fprintf(w, `[
	{
		"name": "weaveworks-nginx",
		"description": "This installs nginx."
	}
]`)
		}))

	server = &http.Server{Addr: ":8080", Handler: mux}

	go func() {
		_ = server.ListenAndServe()
	}()

})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	Expect(server.Shutdown(context.Background())).To(Succeed())
})
