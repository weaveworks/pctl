package git_test

import (
	"errors"
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

	Context("HasChanges", func() {
		When("the flow is disrupted with errors", func() {
			It("returns false and a sensible wrapped error", func() {
				runner.RunReturns([]byte(""), errors.New("nope"))
				g := git.NewCLIGit(git.CLIGitConfig{
					Filename: "filename",
					Location: "location",
					Branch:   "main",
					Remote:   "origin",
				}, runner)
				ok, err := g.HasChanges()
				Expect(err).To(MatchError(`failed to check if there are changes: nope`))
				Expect(ok).To(BeFalse())
				Expect(runner.RunCallCount()).To(Equal(1))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "status", "-s"}))
			})
		})
		When("normal flow operations", func() {
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
				Expect(runner.RunCallCount()).To(Equal(1))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "status", "-s"}))
			})
			It("detects if there are no changes if stats returns empty", func() {
				runner.RunReturns([]byte(""), nil)
				g := git.NewCLIGit(git.CLIGitConfig{
					Filename: "filename",
					Location: "location",
					Branch:   "main",
					Remote:   "origin",
				}, runner)
				ok, err := g.HasChanges()
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
				Expect(runner.RunCallCount()).To(Equal(1))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "status", "-s"}))
			})
		})
	})

	Context("Add", func() {
		When("normal flow operations", func() {
			It("can add changes to a commit", func() {
				g := git.NewCLIGit(git.CLIGitConfig{
					Filename: "filename",
					Location: "location",
					Branch:   "main",
					Remote:   "origin",
				}, runner)
				err := g.Add()
				Expect(err).NotTo(HaveOccurred())
				Expect(runner.RunCallCount()).To(Equal(1))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "add", "filename"}))
			})
		})
		When("the flow is disrupted with errors", func() {
			It("returns a sensible wrapped error", func() {
				runner.RunReturns([]byte(""), errors.New("nope"))
				g := git.NewCLIGit(git.CLIGitConfig{
					Filename: "filename",
					Location: "location",
					Branch:   "main",
					Remote:   "origin",
				}, runner)
				err := g.Add()
				Expect(err).To(MatchError(`failed to run add: nope`))
				Expect(runner.RunCallCount()).To(Equal(1))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "add", "filename"}))
			})
		})
	})

	Context("Push", func() {
		When("normal flow operations", func() {
			It("pushes changes to a remote", func() {
				g := git.NewCLIGit(git.CLIGitConfig{
					Filename: "filename",
					Location: "location",
					Branch:   "main",
					Remote:   "origin",
				}, runner)
				err := g.Push()
				Expect(err).NotTo(HaveOccurred())
				Expect(runner.RunCallCount()).To(Equal(1))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "push", "origin", "main"}))
			})
		})
		When("the flow is disrupted with errors", func() {
			It("returns a sensible wrapped error", func() {
				runner.RunReturns([]byte(""), errors.New("nope"))
				g := git.NewCLIGit(git.CLIGitConfig{
					Filename: "filename",
					Location: "location",
					Branch:   "main",
					Remote:   "origin",
				}, runner)
				err := g.Push()
				Expect(err).To(MatchError(`failed to push changes to remote origin with branch main: nope`))
				Expect(runner.RunCallCount()).To(Equal(1))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "push", "origin", "main"}))
			})
		})
	})

	Context("Commit", func() {
		When("normal flow operations", func() {
			It("commit changes", func() {
				runner.RunReturnsOnCall(0, []byte("profile_subscription.yaml"), nil)
				g := git.NewCLIGit(git.CLIGitConfig{
					Filename: "filename",
					Location: "location",
					Branch:   "main",
					Remote:   "origin",
				}, runner)
				err := g.Commit()
				Expect(err).NotTo(HaveOccurred())
				Expect(runner.RunCallCount()).To(Equal(2))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "status", "-s"}))
				arg, args = runner.RunArgsForCall(1)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "commit", "-m", "Push changes to remote", "filename"}))
			})
		})
		When("the flow is disrupted with errors", func() {
			It("returns a sensible wrapped error", func() {
				runner.RunReturnsOnCall(0, []byte("profile_subscription.yaml"), nil)
				runner.RunReturnsOnCall(1, []byte(""), errors.New("nope"))
				g := git.NewCLIGit(git.CLIGitConfig{
					Filename: "filename",
					Location: "location",
					Branch:   "main",
					Remote:   "origin",
				}, runner)
				err := g.Commit()
				Expect(err).To(MatchError(`failed to run commit: nope`))
				Expect(runner.RunCallCount()).To(Equal(2))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "status", "-s"}))
				arg, args = runner.RunArgsForCall(1)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "commit", "-m", "Push changes to remote", "filename"}))
			})
		})
	})

	Context("CreateBranch", func() {
		When("normal flow operations", func() {
			It("creates a branch if it differs from base", func() {
				g := git.NewCLIGit(git.CLIGitConfig{
					Filename: "filename",
					Location: "location",
					Branch:   "test01",
					Remote:   "origin",
					Base:     "main",
				}, runner)
				err := g.CreateBranch()
				Expect(err).NotTo(HaveOccurred())
				Expect(runner.RunCallCount()).To(Equal(1))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "checkout", "-b", "test01"}))

			})
			It("doesn't do anything if the branch equals the base", func() {
				g := git.NewCLIGit(git.CLIGitConfig{
					Filename: "filename",
					Location: "location",
					Branch:   "main",
					Remote:   "origin",
					Base:     "main",
				}, runner)
				err := g.CreateBranch()
				Expect(err).NotTo(HaveOccurred())
				Expect(runner.RunCallCount()).To(Equal(0))
			})
		})
		When("the flow is disrupted with errors", func() {
			It("returns a sensible wrapped error", func() {
				runner.RunReturns([]byte(""), errors.New("nope"))
				g := git.NewCLIGit(git.CLIGitConfig{
					Filename: "filename",
					Location: "location",
					Branch:   "test01",
					Remote:   "origin",
					Base:     "main",
				}, runner)
				err := g.CreateBranch()
				Expect(err).To(MatchError(`failed to create new branch test01: nope`))
				Expect(runner.RunCallCount()).To(Equal(1))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "checkout", "-b", "test01"}))
			})
		})
	})

	Context("IsRepository", func() {
		When("the flow is disrupted with errors", func() {
			It("return a sensible error", func() {
				g := git.NewCLIGit(git.CLIGitConfig{
					Filename: "filename",
					Location: "notexists",
					Branch:   "main",
					Remote:   "origin",
				}, runner)
				err := g.IsRepository()
				Expect(os.IsNotExist(err)).To(BeTrue())
				Expect(runner.RunCallCount()).To(Equal(0))
			})
		})
		When("normal flow operations", func() {
			It("detects git repositories", func() {
				tmp, err := ioutil.TempDir("", "detect_git_repo_01")
				Expect(err).NotTo(HaveOccurred())
				err = os.Mkdir(filepath.Join(tmp, ".git"), os.ModeDir)
				Expect(err).NotTo(HaveOccurred())
				g := git.NewCLIGit(git.CLIGitConfig{
					Filename: "filename",
					Location: tmp,
					Branch:   "main",
					Remote:   "origin",
				}, runner)
				err = g.IsRepository()
				Expect(err).NotTo(HaveOccurred())
				Expect(runner.RunCallCount()).To(Equal(0))
			})
			It("returns an error when the folder is not a git repository", func() {
				tmp, err := ioutil.TempDir("", "detect_git_repo_02")
				Expect(err).NotTo(HaveOccurred())
				g := git.NewCLIGit(git.CLIGitConfig{
					Filename: "filename",
					Location: tmp,
					Branch:   "main",
					Remote:   "origin",
				}, runner)
				err = g.IsRepository()
				Expect(err).To(HaveOccurred())
				Expect(runner.RunCallCount()).To(Equal(0))
			})
		})
	})
})
