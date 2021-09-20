package bootstrap_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/bootstrap"
	"github.com/weaveworks/pctl/pkg/runner"
	"github.com/weaveworks/pctl/pkg/runner/fakes"
)

var _ = Describe("Bootstrap", func() {
	var (
		temp string
		cfg  bootstrap.Config
	)

	BeforeEach(func() {
		var err error
		temp, err = ioutil.TempDir("", "pctl-bootstrap-test")
		Expect(err).ToNot(HaveOccurred())
		bootstrap.SetRunner(&runner.CLIRunner{})
		cfg = bootstrap.Config{
			GitRepository: profilesv1.GitRepository{
				Name:      "bar",
				Namespace: "foo",
			},
			DefaultDir: "default-dir",
		}
	})

	AfterEach(func() {
		_ = os.RemoveAll(temp)
	})

	Describe("CreateConfig", func() {

		It("creates the config file with the git repository value set", func() {
			cmd := exec.Command("git", "init", temp)
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("init failed: %s", string(output)))

			Expect(bootstrap.CreateConfig(cfg, temp)).To(Succeed())
			data, err := ioutil.ReadFile(filepath.Join(temp, ".pctl", "config.yaml"))
			Expect(err).ToNot(HaveOccurred())

			var config bootstrap.Config
			Expect(yaml.Unmarshal(data, &config)).To(Succeed())
			Expect(config).To(Equal(cfg))
		})
		When("the directory is not a git directory", func() {
			It("returns an error", func() {
				err := bootstrap.CreateConfig(cfg, temp)
				Expect(err).To(MatchError(fmt.Sprintf("the target directory %q is not a git repository", temp)))
			})
		})

		When("it fails to check if its a git repository", func() {
			BeforeEach(func() {
				fakeRunner := new(fakes.FakeRunner)
				bootstrap.SetRunner(fakeRunner)
				fakeRunner.RunReturns([]byte(""), fmt.Errorf("foo"))
			})

			It("returns an error", func() {
				err := bootstrap.CreateConfig(cfg, temp)
				Expect(err).To(MatchError("failed to get git directory location: foo"))
			})
		})
	})

	Describe("GetConfig", func() {
		It("returns the config", func() {
			cmd := exec.Command("git", "init", temp)
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("init failed: %s", string(output)))

			pctlDir := filepath.Join(temp, ".pctl")
			Expect(os.Mkdir(pctlDir, 0755)).To(Succeed())

			data, err := yaml.Marshal(cfg)

			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(filepath.Join(pctlDir, "config.yaml"), data, 0644)).To(Succeed())

			config := bootstrap.GetConfig(temp)
			Expect(*config).To(Equal(cfg))
		})

		When("the directory is not a git directory", func() {
			It("returns an error", func() {
				config := bootstrap.GetConfig(temp)
				Expect(config).To(BeNil())
			})
		})

		When("the config doesn't exist", func() {
			It("returns the config", func() {
				cmd := exec.Command("git", "init", temp)
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("init failed: %s", string(output)))
				config := bootstrap.GetConfig(temp)
				Expect(config).To(BeNil())
			})
		})

		When("it fails to check if its a git repository", func() {
			BeforeEach(func() {
				fakeRunner := new(fakes.FakeRunner)
				bootstrap.SetRunner(fakeRunner)
				fakeRunner.RunReturns([]byte(""), fmt.Errorf("foo"))
			})

			It("returns an error", func() {
				config := bootstrap.GetConfig(temp)
				Expect(config).To(BeNil())
			})
		})

		When("the config isn't valid yaml", func() {
			It("returns the config", func() {
				cmd := exec.Command("git", "init", temp)
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("init failed: %s", string(output)))

				pctlDir := filepath.Join(temp, ".pctl")
				Expect(os.Mkdir(pctlDir, 0755)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(pctlDir, "config.yaml"), []byte("!.z123"), 0644)).To(Succeed())

				config := bootstrap.GetConfig(temp)
				Expect(config).To(BeNil())
			})
		})
	})
})
