package git_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/runner/fakes"
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
					Directory: "location",
					Branch:    "main",
					Remote:    "origin",
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
				runner.RunReturns([]byte("profile_installation.yaml"), nil)
				g := git.NewCLIGit(git.CLIGitConfig{
					Directory: "location",
					Branch:    "main",
					Remote:    "origin",
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
					Directory: "location",
					Branch:    "main",
					Remote:    "origin",
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
					Directory: "location",
					Branch:    "main",
					Remote:    "origin",
				}, runner)
				err := g.Add()
				Expect(err).NotTo(HaveOccurred())
				Expect(runner.RunCallCount()).To(Equal(1))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "add", "."}))
			})
		})
		When("the flow is disrupted with errors", func() {
			It("returns a sensible wrapped error", func() {
				runner.RunReturns([]byte(""), errors.New("nope"))
				g := git.NewCLIGit(git.CLIGitConfig{
					Directory: "location",
					Branch:    "main",
					Remote:    "origin",
				}, runner)
				err := g.Add()
				Expect(err).To(MatchError(`failed to run add: nope`))
				Expect(runner.RunCallCount()).To(Equal(1))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "add", "."}))
			})
		})
	})

	Context("Push", func() {
		When("normal flow operations", func() {
			It("pushes changes to a remote", func() {
				g := git.NewCLIGit(git.CLIGitConfig{
					Directory: "location",
					Branch:    "main",
					Remote:    "origin",
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
					Directory: "location",
					Branch:    "main",
					Remote:    "origin",
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
				runner.RunReturnsOnCall(0, []byte("profile_installation.yaml"), nil)
				g := git.NewCLIGit(git.CLIGitConfig{
					Directory: "location",
					Branch:    "main",
					Remote:    "origin",
				}, runner)
				err := g.Commit()
				Expect(err).NotTo(HaveOccurred())
				Expect(runner.RunCallCount()).To(Equal(2))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "status", "-s"}))
				arg, args = runner.RunArgsForCall(1)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "commit", "-am", "Push changes to remote"}))
			})
		})
		When("the flow is disrupted with errors", func() {
			It("returns a sensible wrapped error", func() {
				runner.RunReturnsOnCall(0, []byte("profile_installation.yaml"), nil)
				runner.RunReturnsOnCall(1, []byte(""), errors.New("nope"))
				g := git.NewCLIGit(git.CLIGitConfig{
					Directory: "location",
					Branch:    "main",
					Remote:    "origin",
				}, runner)
				err := g.Commit()
				Expect(err).To(MatchError(`failed to run commit: nope`))
				Expect(runner.RunCallCount()).To(Equal(2))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "status", "-s"}))
				arg, args = runner.RunArgsForCall(1)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "commit", "-am", "Push changes to remote"}))
			})
		})
	})

	Context("CreateBranch", func() {
		When("normal flow operations", func() {
			It("creates a branch if it differs from base", func() {
				g := git.NewCLIGit(git.CLIGitConfig{
					Directory: "location",
					Branch:    "test01",
					Remote:    "origin",
					Base:      "main",
				}, runner)
				err := g.CreateBranch("test01")
				Expect(err).NotTo(HaveOccurred())
				Expect(runner.RunCallCount()).To(Equal(1))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "checkout", "-b", "test01"}))

			})
			It("doesn't do anything if the branch equals the base", func() {
				g := git.NewCLIGit(git.CLIGitConfig{
					Directory: "location",
					Branch:    "main",
					Remote:    "origin",
					Base:      "main",
				}, runner)
				err := g.CreateBranch("main")
				Expect(err).NotTo(HaveOccurred())
				Expect(runner.RunCallCount()).To(Equal(0))
			})
		})
		When("the flow is disrupted with errors", func() {
			It("returns a sensible wrapped error", func() {
				runner.RunReturns([]byte(""), errors.New("nope"))
				g := git.NewCLIGit(git.CLIGitConfig{
					Directory: "location",
					Branch:    "test01",
					Remote:    "origin",
					Base:      "main",
				}, runner)
				err := g.CreateBranch("test01")
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
					Directory: "notexists",
					Branch:    "main",
					Remote:    "origin",
				}, runner)
				err := g.IsRepository()
				Expect(os.IsNotExist(err)).To(BeTrue())
				Expect(runner.RunCallCount()).To(Equal(0))
			})
		})
		When("normal flow operations", func() {
			var tmp string
			BeforeEach(func() {
				var err error
				tmp, err = ioutil.TempDir("", "detect_git_repo")
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				Expect(os.RemoveAll(tmp)).To(Succeed())
			})

			It("detects git repositories", func() {
				Expect(os.Mkdir(filepath.Join(tmp, ".git"), os.ModeDir)).To(Succeed())
				g := git.NewCLIGit(git.CLIGitConfig{
					Directory: tmp,
					Branch:    "main",
					Remote:    "origin",
				}, runner)
				Expect(g.IsRepository()).To(Succeed())
				Expect(runner.RunCallCount()).To(Equal(0))
			})
			It("returns an error when the folder is not a git repository", func() {
				g := git.NewCLIGit(git.CLIGitConfig{
					Directory: tmp,
					Branch:    "main",
					Remote:    "origin",
				}, runner)
				Expect(g.IsRepository()).NotTo(Succeed())
				Expect(runner.RunCallCount()).To(Equal(0))
			})
		})
	})

	Context("Init", func() {
		var tmp string
		BeforeEach(func() {
			var err error
			tmp, err = ioutil.TempDir("", "detect_git_repo")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tmp)).To(Succeed())
		})

		It("initiatest the git repository", func() {
			g := git.NewCLIGit(git.CLIGitConfig{
				Directory: tmp,
				Branch:    "main",
				Remote:    "origin",
			}, runner)
			Expect(g.Init()).To(Succeed())
			Expect(runner.RunCallCount()).To(Equal(1))
			arg, args := runner.RunArgsForCall(0)
			Expect(arg).To(Equal("git"))
			Expect(args).To(Equal([]string{"init", tmp, "-b", "main"}))
		})

		When("init fails", func() {
			It("returns an error", func() {
				runner.RunReturns(nil, fmt.Errorf("init failed"))
				g := git.NewCLIGit(git.CLIGitConfig{
					Directory: tmp,
					Branch:    "main",
					Remote:    "origin",
				}, runner)
				Expect(g.Init()).To(MatchError("failed to init: init failed"))
				Expect(runner.RunCallCount()).To(Equal(1))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"init", tmp, "-b", "main"}))
			})
		})
	})

	Context("Merge", func() {
		It("merges the branch", func() {
			g := git.NewCLIGit(git.CLIGitConfig{
				Directory: "location",
				Branch:    "main",
				Remote:    "origin",
			}, runner)
			mergeConflict, err := g.Merge("user-changes")
			Expect(err).NotTo(HaveOccurred())
			Expect(mergeConflict).To(BeFalse())
			Expect(runner.RunCallCount()).To(Equal(1))
			arg, args := runner.RunArgsForCall(0)
			Expect(arg).To(Equal("git"))
			Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "merge", "user-changes"}))
		})

		When("an error occurings during merge", func() {
			When("its not due to a conflict", func() {
				It("returns an error", func() {
					runner.RunReturns(nil, fmt.Errorf("merge failed"))
					g := git.NewCLIGit(git.CLIGitConfig{
						Directory: "location",
						Branch:    "main",
						Remote:    "origin",
					}, runner)
					mergeConflict, err := g.Merge("user-changes")
					Expect(err).To(MatchError("failed to run merge: merge failed"))
					Expect(mergeConflict).To(BeFalse())
					Expect(runner.RunCallCount()).To(Equal(1))
					arg, args := runner.RunArgsForCall(0)
					Expect(arg).To(Equal("git"))
					Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "merge", "user-changes"}))
				})
			})

			When("its due to a conflict", func() {
				It("returns conflict true and no error", func() {
					runner.RunReturns([]byte("some text followed by: Merge conflict"), fmt.Errorf("merge failed"))
					g := git.NewCLIGit(git.CLIGitConfig{
						Directory: "location",
						Branch:    "main",
						Remote:    "origin",
					}, runner)
					mergeConflict, err := g.Merge("user-changes")
					Expect(err).NotTo(HaveOccurred())
					Expect(mergeConflict).To(BeTrue())
					Expect(runner.RunCallCount()).To(Equal(1))
					arg, args := runner.RunArgsForCall(0)
					Expect(arg).To(Equal("git"))
					Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "merge", "user-changes"}))
				})
			})
		})
	})

	Context("Checkout", func() {
		It("checkouts the branch", func() {
			g := git.NewCLIGit(git.CLIGitConfig{
				Directory: "location",
				Branch:    "test01",
				Remote:    "origin",
				Base:      "main",
			}, runner)
			err := g.Checkout("test01")
			Expect(err).NotTo(HaveOccurred())
			Expect(runner.RunCallCount()).To(Equal(1))
			arg, args := runner.RunArgsForCall(0)
			Expect(arg).To(Equal("git"))
			Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "checkout", "test01"}))
		})

		When("an error occurs", func() {
			It("returns an error", func() {
				runner.RunReturns([]byte(""), errors.New("nope"))
				g := git.NewCLIGit(git.CLIGitConfig{
					Directory: "location",
					Branch:    "test01",
					Remote:    "origin",
					Base:      "main",
				}, runner)
				err := g.Checkout("test01")
				Expect(err).To(MatchError(`failed to checkout branch test01: nope`))
				Expect(runner.RunCallCount()).To(Equal(1))
				arg, args := runner.RunArgsForCall(0)
				Expect(arg).To(Equal("git"))
				Expect(args).To(Equal([]string{"--git-dir", "location/.git", "--work-tree", "location", "checkout", "test01"}))
			})
		})
	})
})
