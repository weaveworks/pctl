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

var _ = Describe("bootstrap", func() {
	BeforeEach(func() {
		var err error
		temp, err = ioutil.TempDir("", "kivo-bootstrap-test")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		_ = os.RemoveAll(temp)
	})

	When("passing the directory in as an argument", func() {
		It("creates the kivo configuration file", func() {
			cmd := exec.Command("git", "init", temp)
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("init failed: %s", string(output)))

			Expect(kivo("bootstrap", "--git-repository", "foo/bar", "--default-dir", "default-dir", temp)).To(ContainElement("✔ bootstrap completed"))
			data, err := ioutil.ReadFile(filepath.Join(temp, ".kivo", "config.yaml"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(`gitRepository:
  name: bar
  namespace: foo
defaultDir: default-dir
`))
		})
	})

	When("not providing the directory", func() {
		It("creates the kivo configuration file in your working directory", func() {
			cmd := exec.Command("git", "init", temp)
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("init failed: %s", string(output)))

			Expect(kivo("bootstrap", "--git-repository", "foo/bar", "--default-dir", "default-dir")).To(ContainElement("✔ bootstrap completed"))
			data, err := ioutil.ReadFile(filepath.Join(temp, ".kivo", "config.yaml"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(`gitRepository:
  name: bar
  namespace: foo
defaultDir: default-dir
`))
		})
	})

	When("passing a relative path", func() {
		It("creates the kivo configuration file in correct directory", func() {
			cmd := exec.Command("git", "init", temp)
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("init failed: %s", string(output)))

			Expect(kivo("bootstrap", "--git-repository", "foo/bar", "--default-dir", "default-dir", ".")).To(ContainElement("✔ bootstrap completed"))
			data, err := ioutil.ReadFile(filepath.Join(temp, ".kivo", "config.yaml"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(`gitRepository:
  name: bar
  namespace: foo
defaultDir: default-dir
`))
		})
	})
})
