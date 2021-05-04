package integration_test

import (
	"fmt"
	"os/exec"
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

	// Set up the test environment using prepare, then patch the image to localhost.
	cmd := exec.Command(binaryPath, "prepare")
	output, err := cmd.CombinedOutput()
	Expect(err).ToNot(HaveOccurred())
	Expect(string(output)).To(Equal("install finished\n"))
	fmt.Println("Installation done successfully.")
	fmt.Print("Waiting for deployment...")
	// Adding waiters
	args := []string{
		"-n",
		"profiles-system",
		"wait",
		"--for=condition=available",
		"deployment",
		"profiles-controller-manager",
		"--timeout",
		"5m",
	}
	cmd = exec.Command("kubectl", args...)
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error from kubectl: ", string(output))
	}
	Expect(err).NotTo(HaveOccurred())
	fmt.Println("done.")
	fmt.Print("Waiting for pods to be active...")
	// wait for pods
	args = []string{
		"-n",
		"profiles-system",
		"wait",
		"--for=condition=Ready",
		"--all",
		"pods",
		"--timeout",
		"5m",
	}
	cmd = exec.Command("kubectl", args...)
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error from kubectl: ", string(output))
	}
	Expect(err).NotTo(HaveOccurred())
	fmt.Println("done.")
	fmt.Print("Applying test catalog...")
	//	kubectl apply -f dependencies/profiles/examples/profile-catalog-source.yaml
	args = []string{
		"apply",
		"-f",
		"dependencies/profiles/examples/profile-catalog-source.yaml",
	}
	cmd = exec.Command("kubectl", args...)
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error from kubectl: ", string(output))
	}
	Expect(err).NotTo(HaveOccurred())
	fmt.Println("done. Happy testing!")
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
