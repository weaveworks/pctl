package acceptance_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
)

var (
	binaryPath                      string
	skipTestsThatRequireCredentials bool
)

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Acceptance Suite")
}

var (
	kClient client.Client
)

var _ = BeforeSuite(func() {
	var err error
	binaryPath, err = gexec.Build("github.com/weaveworks/pctl/cmd/pctl")
	Expect(err).NotTo(HaveOccurred())

	// overwrite the default test repository location if set
	if v := os.Getenv("PCTL_TEST_REPOSITORY_URL"); v != "" {
		pctlTestRepositoryName = v
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
	Expect(prepareTestCluster(binaryPath)).To(Succeed())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

// prepareTestCluster will create a test cluster using pctl `prepare` command.
// @binary -- location of the built pctl binary.
func prepareTestCluster(binaryPath string) error {
	var (
		waitForDeploymentArgs = []string{
			"-n",
			"profiles-system",
			"wait",
			"--for=condition=available",
			"deployment",
			"profiles-controller-manager",
			"--timeout",
			"5m",
		}
		waitForPodsArgs = []string{
			"-n",
			"profiles-system",
			"wait",
			"--for=condition=Ready",
			"--all",
			"pods",
			"--timeout",
			"5m",
		}
		applySourceCatalogArgs = []string{
			"apply",
			"-f",
			"catalog-source.yaml",
		}
	)

	tmp, err := ioutil.TempDir("", "prepare_integration_test_01")
	if err != nil {
		return fmt.Errorf("failed to create temp folder for test: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmp); err != nil {
			fmt.Printf("failed to remove temporary folder at location: %s. Please clean manually.", tmp)
		}
	}()
	cmd := exec.Command(binaryPath, "prepare", "--dry-run", "--out", tmp, "--keep")
	output, err := cmd.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte("kind: List")) {
		fmt.Println("Output of prepare was: ", string(output))
		return fmt.Errorf("failed to run prepare command: %w", err)
	}
	fmt.Println("Install file generated successfully.")

	content, err := ioutil.ReadFile(filepath.Join(tmp, "prepare.yaml"))
	if err != nil {
		return fmt.Errorf("failed to read prepare.yaml from location %s: %w", tmp, err)
	}
	fmt.Print("Replacing controller image to localhost:5000...")
	re := regexp.MustCompile(`weaveworks/profiles-controller:.*`)
	out := re.ReplaceAllString(string(content), "localhost:5000/profiles-controller:latest")
	fmt.Println("done.")

	fmt.Print("Applying modified prepare.yaml...")
	applyPrepareArgs := []string{"apply", "-f", "-"}
	cmd = exec.Command("kubectl", applyPrepareArgs...)
	in, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get in pipe for kubectl: %w", err)
	}

	go func() {
		defer func(in io.WriteCloser) {
			_ = in.Close()
		}(in)
		if _, err := io.WriteString(in, out); err != nil {
			fmt.Println("Failed to write to kubectl apply: ", err)
		}
	}()

	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println("\nOutput from kubectl apply: ", string(output))
		return fmt.Errorf("failed to apply prepare yaml: %w", err)
	}
	fmt.Println("done.")

	fmt.Print("Waiting for deployment...")
	if err := runKubectl(waitForDeploymentArgs); err != nil {
		return fmt.Errorf("failed to wait for deployment: %w", err)
	}

	fmt.Print("Waiting for pods to be active...")
	if err := runKubectl(waitForPodsArgs); err != nil {
		return fmt.Errorf("failed to wait for pods to be active: %w", err)
	}

	fmt.Print("Applying test catalog...")
	if err := runKubectl(applySourceCatalogArgs); err != nil {
		return fmt.Errorf("failed to apply test catalog: %w", err)
	}

	fmt.Println("Happy testing!")
	return nil
}

func runKubectl(args []string) error {
	cmd := exec.Command("kubectl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error from kubectl: ", string(output))
		return err
	}
	fmt.Println("done.")
	return nil
}
