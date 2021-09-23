package integration_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/kivo-cli/tests/integration"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var (
	binaryPath                      string
	skipTestsThatRequireCredentials bool
	kivoTestRepositoryName          = "git@github.com:weaveworks/pctl-test-repo.git"
	// used when creating a pr
	kivoTestRepositoryOrgName = "weaveworks/pctl-test-repo"
	// used for flux repository branch creation
	kivoTestRepositoryHTTP = "https://github.com/weaveworks/pctl-test-repo"
	kClient                client.Client
	temp                   string
	namespace              string
	configMapName          string
	branch                 string
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func() {
	var err error
	binaryPath, err = gexec.Build("github.com/weaveworks/kivo-cli/cmd/kivo")
	Expect(err).NotTo(HaveOccurred())

	// overwrite the default test repository location if set
	if v := os.Getenv("PCTL_TEST_REPOSITORY_URL"); v != "" {
		kivoTestRepositoryName = v
		repoName := path.Base(kivoTestRepositoryName)
		repoName = strings.TrimSuffix(repoName, ".git")
		re := regexp.MustCompile("^(https|git)(://|@)([^/:]+)[/:]([^/:]+)/(.+)$")
		m := re.FindAllStringSubmatch(kivoTestRepositoryName, -1)
		if len(m) == 0 || len(m[0]) < 5 {
			Fail("failed to extract repo user from the url, only github with https or git format is supported atm")
		}
		repoUser := m[0][4]
		kivoTestRepositoryHTTP = fmt.Sprintf("https://github.com/%s/%s", repoUser, repoName)
		kivoTestRepositoryOrgName = fmt.Sprintf("%s/%s", repoUser, repoName)
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

var _ = BeforeEach(func() {
	var err error
	temp, err = ioutil.TempDir("", "kivo_tmp")
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterEach(func() {
	_ = os.RemoveAll(temp)
})

func kivo(args ...string) []string {
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = temp
	session, err := cmd.CombinedOutput()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), fmt.Sprintf("error occurred running: kivo %s. Output: %s", strings.Join(args, " "), string(session)))
	return sanitiseString(string(session))
}

func kivoWithError(args ...string) []string {
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = temp
	session, err := cmd.CombinedOutput()
	ExpectWithOffset(1, err).To(HaveOccurred(), fmt.Sprintf("error occurred running: kivo %s. Output: %s", strings.Join(args, " "), string(session)))

	return sanitiseString(string(session))
}

func kivoWithRawOutput(args ...string) string {
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = temp
	session, err := cmd.CombinedOutput()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), fmt.Sprintf("error occurred running: kivo %s. Output: %s", strings.Join(args, " "), string(session)))
	return string(session)
}

func sanitiseString(session string) []string {
	session = strings.Replace(session, "\t", " ", -1)
	session = strings.TrimSuffix(session, "\n")
	parts := strings.Split(session, "\n")

	var newParts []string
	for _, part := range parts {
		if part != "" {
			newParts = append(newParts, strings.TrimSpace(part))
		}
	}

	return newParts
}

func filesInDir(profileDir string) []string {
	var files []string
	err := filepath.Walk(profileDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, strings.TrimPrefix(path, profileDir+"/"))
		}
		return nil
	})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return files
}

func catFile(filename string) string {
	content, err := ioutil.ReadFile(filename)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return string(content)
}

func cloneAndCheckoutBranch(temp, branch string) {
	// check out the branch
	cmd := exec.Command("git", "clone", kivoTestRepositoryName, temp)
	output, err := cmd.CombinedOutput()
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), fmt.Sprintf("clone failed: %s", string(output)))
	cmd = exec.Command("git", "--git-dir", filepath.Join(temp, ".git"), "--work-tree", temp, "checkout", "-b", branch)
	output, err = cmd.CombinedOutput()
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), fmt.Sprintf("checkout branch failed: %s", string(output)))

	cmd = exec.Command("git", "--git-dir", filepath.Join(temp, ".git"), "--work-tree", temp, "push", "-u", "origin", branch)
	output, err = cmd.CombinedOutput()
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), fmt.Sprintf("git push failed : %s", string(output)))
	// setup the gitrepository resources. Requires the branch to exist first
}

func gitAddAndPush(dir, branch string) {
	cmd := exec.Command("git", "--git-dir", filepath.Join(dir, ".git"), "--work-tree", dir, "add", ".")
	output, err := cmd.CombinedOutput()
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), fmt.Sprintf("git add . failed: %s", string(output)))
	cmd = exec.Command("git", "--git-dir", filepath.Join(dir, ".git"), "--work-tree", dir, "commit", "-am", "new content")
	output, err = cmd.CombinedOutput()
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), fmt.Sprintf("git commit failed : %s", string(output)))
	cmd = exec.Command("git", "--git-dir", filepath.Join(dir, ".git"), "--work-tree", dir, "push", "-u", "origin", branch)
	output, err = cmd.CombinedOutput()
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), fmt.Sprintf("git push failed : %s", string(output)))
}

func createNamespace(namespace string) {
	nsp := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	ExpectWithOffset(1, kClient.Create(context.Background(), &nsp)).To(Succeed())
}
func deleteNamespace(namespace string) {
	nsp := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	_ = kClient.Delete(context.Background(), &nsp)
	EventuallyWithOffset(1, func() error {
		err := kClient.Get(context.Background(), client.ObjectKey{Name: namespace}, &nsp)
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("delete not finished yet: %w", err)
	}, "2m", "1s").Should(Succeed())
}
