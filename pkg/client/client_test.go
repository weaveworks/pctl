package client_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/client"
)

var _ = Describe("Client", func() {
	Describe("NewFromOptions", func() {
		When("the kubeconfig doesn't exist", func() {
			It("returns an error", func() {
				_, err := client.NewFromOptions(client.ServiceOptions{
					KubeconfigPath: "/i/dont/exists",
				})
				Expect(err).To(MatchError(ContainSubstring("failed to create config from kubeconfig path")))
			})
		})
	})
})
