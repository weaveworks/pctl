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

	"github.com/weaveworks/pctl/pkg/bootstrap"
	"github.com/weaveworks/pctl/pkg/runner"
	"github.com/weaveworks/pctl/pkg/runner/fakes"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

var _ = Describe("Bootstrap", func() {
	var (
		temp string
	)

	BeforeEach(func() {
		var err error
		temp, err = ioutil.TempDir("", "pctl-bootstrap-test")
		Expect(err).ToNot(HaveOccurred())
		bootstrap.SetRunner(&runner.CLIRunner{})
	})

	AfterEach(func() {
		_ = os.RemoveAll(temp)
	})

	Describe("CreateConfig", func() {
		It("creates the config file with the git repository value set", func() {
			cmd := exec.Command("git", "init", temp)
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("init failed: %s", string(output)))

			Expect(bootstrap.CreateConfig("foo", "bar", temp)).To(Succeed())
			data, err := ioutil.ReadFile(filepath.Join(temp, ".pctl", "config.yaml"))
			Expect(err).ToNot(HaveOccurred())

			var config bootstrap.Config
			Expect(yaml.Unmarshal(data, &config)).To(Succeed())
			Expect(config).To(Equal(bootstrap.Config{
				GitRepository: profilesv1.GitRepository{
					Name:      "bar",
					Namespace: "foo",
				},
			}))
		})

		When("the directory is not a git directory", func() {
			It("returns an error", func() {
				err := bootstrap.CreateConfig("foo", "bar", temp)
				Expect(err).To(MatchError(fmt.Sprintf("the target directory %q is not a git repository", temp)))
			})
		})

		When("the directory is not a git directory", func() {
			BeforeEach(func() {
				fakeRunner := new(fakes.FakeRunner)
				bootstrap.SetRunner(fakeRunner)
				fakeRunner.RunReturns([]byte(""), fmt.Errorf("foo"))
			})

			It("returns an error", func() {
				err := bootstrap.CreateConfig("foo", "bar", temp)
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

			data, err := yaml.Marshal(bootstrap.Config{
				GitRepository: profilesv1.GitRepository{
					Name:      "foo",
					Namespace: "bar",
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(filepath.Join(pctlDir, "config.yaml"), data, 0644)).To(Succeed())

			config, err := bootstrap.GetConfig(temp)
			Expect(err).NotTo(HaveOccurred())
			Expect(*config).To(Equal(bootstrap.Config{
				GitRepository: profilesv1.GitRepository{
					Name:      "foo",
					Namespace: "bar",
				},
			}))
		})

		When("the directory is not a git directory", func() {
			It("returns an error", func() {
				_, err := bootstrap.GetConfig(temp)
				Expect(err).To(MatchError(fmt.Sprintf("the target directory %q is not a git repository", temp)))
			})
		})

		When("the config doesn't exist", func() {
			It("returns the config", func() {
				cmd := exec.Command("git", "init", temp)
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("init failed: %s", string(output)))
				_, err = bootstrap.GetConfig(temp)
				Expect(err).To(MatchError(ContainSubstring("failed to read config file")))
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

				_, err = bootstrap.GetConfig(temp)
				Expect(err).To(MatchError(ContainSubstring("failed to unmarshal config file")))
			})
		})
	})
})
