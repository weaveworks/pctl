package integration_test

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
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
	binaryPath                      string
	skipTestsThatRequireCredentials bool
	pctlTestRepositoryName          = "git@github.com:weaveworks/pctl-test-repo.git"
	// used when creating a pr
	pctlTestRepositoryOrgName = "weaveworks/pctl-test-repo"
	// used for flux repository branch creation
	pctlTestRepositoryHTTP = "https://github.com/weaveworks/pctl-test-repo"
	kClient                client.Client
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func() {
	var err error
	binaryPath, err = gexec.Build("github.com/weaveworks/pctl/cmd/pctl")
	Expect(err).NotTo(HaveOccurred())

	// overwrite the default test repository location if set
	if v := os.Getenv("PCTL_TEST_REPOSITORY_URL"); v != "" {
		pctlTestRepositoryName = v
		repoName := path.Base(pctlTestRepositoryName)
		repoName = strings.TrimSuffix(repoName, ".git")
		re := regexp.MustCompile("^(https|git)(://|@)([^/:]+)[/:]([^/:]+)/(.+)$")
		m := re.FindAllStringSubmatch(pctlTestRepositoryName, -1)
		if len(m) == 0 || len(m[0]) < 5 {
			Fail("failed to extract repo user from the url, only github with https or git format is supported atm")
		}
		repoUser := m[0][4]
		pctlTestRepositoryHTTP = fmt.Sprintf("https://github.com/%s/%s", repoUser, repoName)
		pctlTestRepositoryOrgName = fmt.Sprintf("%s/%s", repoUser, repoName)
	}

	if v := os.Getenv("SKIP_CREDENTIAL_TESTS"); v == "true" {
		skipTestsThatRequireCredentials = true
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
	err = integration.InstallClusterComponents(binaryPath)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
