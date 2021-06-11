package integration_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/tests/integration"
)

var (
	binaryPath string
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var kClient client.Client

var _ = BeforeSuite(func() {
	var err error
	binaryPath, err = gexec.Build("github.com/weaveworks/pctl/cmd/pctl")
	Expect(err).NotTo(HaveOccurred())

	// overwrite the default test repository location if set
	if v := os.Getenv("PCTL_TEST_REPOSITORY_URL"); v != "" {
		pctlTestRepositoryName = v
	}

	scheme := runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
	Expect(profilesv1.AddToScheme(scheme)).To(Succeed())
	Expect(helmv2.AddToScheme(scheme)).To(Succeed())
	Expect(kustomizev1.AddToScheme(scheme)).To(Succeed())
	Expect(sourcev1.AddToScheme(scheme)).To(Succeed())

	kubeconfig := ctrl.GetConfigOrDie()
	kClient, err = client.New(kubeconfig, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	err = integration.PrepareTestCluster(binaryPath)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
