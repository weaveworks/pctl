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

var _ = Describe("Kivo", func() {
	Context("docgen", func() {
		It("writes kivo command help to markdown files", func() {
			cmd := exec.Command(binaryPath, "docgen", "--path", temp)
			_, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())

			files, err := ioutil.ReadDir(temp)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(files)).To(Equal(5))
			commands := []string{"add", "bootstrap", "install", "get", "upgrade"}
			for _, cmd := range commands {
				filename := filepath.Join(temp, fmt.Sprintf("kivo-%s-cmd.md", cmd))
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
				newDir := filepath.Join(temp, "does-not-exist")
				Expect(newDir).ToNot(BeAnExistingFile())

				cmd := exec.Command(binaryPath, "docgen", "--path", newDir)
				_, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				Expect(newDir).To(BeAnExistingFile())
			})

			When("creating the docs dir fails", func() {
				It("exits 1", func() {
					newDir := filepath.Join(temp, "does-not-exist")
					Expect(os.Chmod(temp, 0600)).To(Succeed())

					cmd := exec.Command(binaryPath, "docgen", "--path", newDir)
					_, err := cmd.CombinedOutput()
					Expect(err).To(HaveOccurred())
				})
			})
		})

		When("writing the docs files fails", func() {
			It("exits 1", func() {
				Expect(os.Chmod(temp, 0600)).To(Succeed())

				cmd := exec.Command(binaryPath, "docgen", "--path", temp)
				_, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
