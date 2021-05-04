package integration

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

const patchController = `{"spec":{"template":{"spec":{"containers":[{"name":"manager","image":"localhost:5000/profiles-controller:latest"}]}}}}`

var (
	patchArgs = []string{
		"-n",
		"profiles-system",
		"patch",
		"deployment",
		"profiles-controller-manager",
		"--patch",
		patchController,
	}
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
		filepath.Join("..", "..", "dependencies/profiles/examples/profile-catalog-source.yaml"),
	}
)

// PrepareTestCluster will create a test cluster using pctl `prepare` command.
// @binary -- location of the built pctl binary.
func PrepareTestCluster(binaryPath string) error {
	cmd := exec.Command(binaryPath, "prepare")
	output, err := cmd.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte("install finished")) {
		fmt.Println("Output of prepare was: ", string(output))
		return fmt.Errorf("failed to run prepare command: %w", err)
	}
	fmt.Println("Installation done successfully.")

	fmt.Print("Patching controller image to localhost:5000...")
	if err := runKubectl(patchArgs); err != nil {
		return fmt.Errorf("failed to apply patch: %w", err)
	}

	fmt.Print("Waiting for deployment...")
	if err := runKubectl(waitForDeploymentArgs); err != nil {
		return fmt.Errorf("failed to wait for deployment: %w", err)
	}

	fmt.Print("Waiting for pods to be active...")
	if err := runKubectl(waitForPodsArgs); err != nil {
		return fmt.Errorf("failed to wait for pods to be active: %w", err)
	}

	// TODO remove this: sleep for testing why the pod wait isn't enough
	time.Sleep(2 * time.Second)

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
