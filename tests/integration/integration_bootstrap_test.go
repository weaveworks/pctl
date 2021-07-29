package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("pctl list", func() {
	var (
		temp string
	)

	BeforeEach(func() {
		var err error
		temp, err = ioutil.TempDir("", "pctl-bootstrap-test")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		_ = os.RemoveAll(temp)
	})

	When("passing the directory in as an argument", func() {
		It("creates the pctl configuration file", func() {
			cmd := exec.Command("git", "init", temp)
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("init failed: %s", string(output)))

			cmd = exec.Command(binaryPath, "bootstrap", "--git-repository", "foo/bar", temp)
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("bootstrap failed: %s", string(session)))
			Expect(string(session)).To(ContainSubstring("bootstrap completed"))
			data, err := ioutil.ReadFile(filepath.Join(temp, ".pctl", "config.yaml"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(`gitrepository:
  name: bar
  namespace: foo
`))
		})
	})

	When("not providing the directory", func() {
		It("creates the pctl configuration file in your working directory", func() {
			cmd := exec.Command("git", "init", temp)
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("init failed: %s", string(output)))

			cmd = exec.Command(binaryPath, "bootstrap", "--git-repository", "foo/bar")
			cmd.Dir = temp
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("bootstrap failed: %s", string(session)))
			Expect(string(session)).To(ContainSubstring("bootstrap completed"))
			data, err := ioutil.ReadFile(filepath.Join(temp, ".pctl", "config.yaml"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(`gitrepository:
  name: bar
  namespace: foo
`))
		})
	})

	When("passing a relative path", func() {
		It("creates the pctl configuration file in correct directory", func() {
			cmd := exec.Command("git", "init", temp)
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("init failed: %s", string(output)))

			cmd = exec.Command(binaryPath, "bootstrap", "--git-repository", "foo/bar", ".")
			cmd.Dir = temp
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("bootstrap failed: %s", string(session)))
			Expect(string(session)).To(ContainSubstring("bootstrap completed"))
			data, err := ioutil.ReadFile(filepath.Join(temp, ".pctl", "config.yaml"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(`gitrepository:
  name: bar
  namespace: foo
`))
		})
	})
})
