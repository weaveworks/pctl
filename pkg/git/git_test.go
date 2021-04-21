package git_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/git/fakes"
)

var _ = Describe("git", func() {
	var (
		runner *fakes.FakeRunner
	)

	BeforeEach(func() {
		runner = new(fakes.FakeRunner)
	})

	When("normal flow operations", func() {
		It("can add changes to a commit", func() {
			g := git.NewCLIGit(git.CLIGitConfig{
				Location: "location",
				Branch:   "main",
				Remote:   "origin",
			}, runner)
			err := g.Add()
			Expect(err).NotTo(HaveOccurred())
		})
		It("commit changes", func() {
			g := git.NewCLIGit(git.CLIGitConfig{
				Location: "location",
				Branch:   "main",
				Remote:   "origin",
			}, runner)
			err := g.Commit()
			Expect(err).NotTo(HaveOccurred())
		})
		It("push changes to a remote", func() {
			g := git.NewCLIGit(git.CLIGitConfig{
				Location: "location",
				Branch:   "main",
				Remote:   "origin",
			}, runner)
			err := g.Push()
			Expect(err).NotTo(HaveOccurred())
		})
		It("detects git repositories", func() {
			tmp, err := ioutil.TempDir("", "detect_git_repo_01")
			Expect(err).NotTo(HaveOccurred())
			err = os.Mkdir(filepath.Join(tmp, ".git"), os.ModeDir)
			Expect(err).NotTo(HaveOccurred())
			g := git.NewCLIGit(git.CLIGitConfig{
				Location: tmp,
				Branch:   "main",
				Remote:   "origin",
			}, runner)
			err = g.IsRepository()
			Expect(err).NotTo(HaveOccurred())
		})
		It("returns an error when the folder is not a git repository", func() {
			tmp, err := ioutil.TempDir("", "detect_git_repo_02")
			Expect(err).NotTo(HaveOccurred())
			g := git.NewCLIGit(git.CLIGitConfig{
				Location: tmp,
				Branch:   "main",
				Remote:   "origin",
			}, runner)
			err = g.IsRepository()
			Expect(err).To(HaveOccurred())
		})
		It("detects if there are changes to be committed if stats returns a list of files", func() {
			runner.RunReturns([]byte("profile_subscription.yaml"), nil)
			g := git.NewCLIGit(git.CLIGitConfig{
				Location: "location",
				Branch:   "main",
				Remote:   "origin",
			}, runner)
			ok, err := g.HasChanges()
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
		})
		It("detects if there are no changes if stats returns empty", func() {
			runner.RunReturns([]byte(""), nil)
			g := git.NewCLIGit(git.CLIGitConfig{
				Location: "location",
				Branch:   "main",
				Remote:   "origin",
			}, runner)
			ok, err := g.HasChanges()
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

	})
})
