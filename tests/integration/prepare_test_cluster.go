package integration

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
)

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
		filepath.Join("..", "..", "dependencies/profiles/examples/profile-catalog-source.yaml"),
	}
)

// PrepareTestCluster will create a test cluster using pctl `prepare` command.
// @binary -- location of the built pctl binary.
func PrepareTestCluster(binaryPath string) error {
	cmd := exec.Command(binaryPath, "prepare", "--dry-run")
	output, err := cmd.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte("kind: List")) {
		fmt.Println("Output of prepare was: ", string(output))
		return fmt.Errorf("failed to run prepare command: %w", err)
	}
	fmt.Println("Install file generated successfully.")

	fmt.Print("Replacing controller image to localhost:5000...")
	re := regexp.MustCompile(`weaveworks/profiles-controller:.*`)
	out := re.ReplaceAllString(string(output), "localhost:5000/profiles-controller:latest")
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
		fmt.Println("Output from kubectl apply: ", string(output))
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
