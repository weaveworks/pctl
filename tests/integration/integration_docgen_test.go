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

var _ = Describe("PCTL", func() {
	Context("docgen", func() {
		var tmpDir string

		BeforeEach(func() {
			var err error
			tmpDir, err = ioutil.TempDir("", "docgen")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			_ = os.RemoveAll(tmpDir)
		})

		It("writes pctl command help to markdown files", func() {
			cmd := exec.Command(binaryPath, "docgen", "--path", tmpDir)
			_, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())

			files, err := ioutil.ReadDir(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(files)).To(Equal(6))
			commands := []string{"install", "prepare", "list", "show", "search", "upgrade"}
			for _, cmd := range commands {
				filename := filepath.Join(tmpDir, fmt.Sprintf("pctl-%s-cmd.md", cmd))
				Expect(filename).To(BeAnExistingFile())
				contents, err := ioutil.ReadFile(filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(ContainSubstring(cmd))
				Expect(string(contents)).To(ContainSubstring("NAME"))
				Expect(string(contents)).To(ContainSubstring("USAGE"))
			}
		})

		When("the provided output directory does not exist", func() {
			It("creates it", func() {
				newDir := filepath.Join(tmpDir, "does-not-exist")
				Expect(newDir).ToNot(BeAnExistingFile())

				cmd := exec.Command(binaryPath, "docgen", "--path", newDir)
				_, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				Expect(newDir).To(BeAnExistingFile())
			})

			When("creating the docs dir fails", func() {
				It("exits 1", func() {
					newDir := filepath.Join(tmpDir, "does-not-exist")
					Expect(os.Chmod(tmpDir, 0600)).To(Succeed())

					cmd := exec.Command(binaryPath, "docgen", "--path", newDir)
					_, err := cmd.CombinedOutput()
					Expect(err).To(HaveOccurred())
				})
			})
		})

		When("writing the docs files fails", func() {
			It("exits 1", func() {
				Expect(os.Chmod(tmpDir, 0600)).To(Succeed())

				cmd := exec.Command(binaryPath, "docgen", "--path", tmpDir)
				_, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
