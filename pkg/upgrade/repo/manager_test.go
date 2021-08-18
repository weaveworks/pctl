package repo_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/git/fakes"
	"github.com/weaveworks/pctl/pkg/upgrade/repo"
)

var _ = Describe("Manager", func() {
	var (
		fakeGitClient *fakes.FakeGit
		manager       repo.RepoManager
	)
	BeforeEach(func() {
		fakeGitClient = new(fakes.FakeGit)
		manager = repo.NewManager(fakeGitClient)
	})

	Describe("CreateRepoWithContents", func() {
		It("creates a repository with the contents", func() {
			called := false
			Expect(manager.CreateRepoWithContent(func() error {
				called = true
				return nil
			})).To(Succeed())

			Expect(called).To(BeTrue())
			Expect(fakeGitClient.InitCallCount()).To(Equal(1))
			Expect(fakeGitClient.AddCallCount()).To(Equal(1))
			Expect(fakeGitClient.AddArgsForCall(0)).To(Equal("."))
			Expect(fakeGitClient.CommitCallCount()).To(Equal(1))
		})

		When("init fails", func() {
			BeforeEach(func() {
				fakeGitClient.InitReturns(fmt.Errorf("failed"))
			})

			It("returns an error", func() {
				Expect(manager.CreateRepoWithContent(func() error {
					return nil
				})).To(MatchError("failed to init repo: failed"))
			})
		})

		When("write content fails", func() {
			It("returns an error", func() {
				Expect(manager.CreateRepoWithContent(func() error {
					return fmt.Errorf("failed")
				})).To(MatchError("failed to write content to repo: failed"))
			})
		})

		When("add fails", func() {
			BeforeEach(func() {
				fakeGitClient.AddReturns(fmt.Errorf("failed"))
			})

			It("returns an error", func() {
				Expect(manager.CreateRepoWithContent(func() error {
					return nil
				})).To(MatchError("failed to add: failed"))
			})
		})

		When("commit fails", func() {
			BeforeEach(func() {
				fakeGitClient.CommitReturns(fmt.Errorf("failed"))
			})

			It("returns an error", func() {
				Expect(manager.CreateRepoWithContent(func() error {
					return nil
				})).To(MatchError("failed to commit: failed"))
			})
		})
	})

	Describe("CreateBranchWithContentFromMain", func() {
		It("creates a repository with the contents", func() {
			called := false
			Expect(manager.CreateBranchWithContentFromMain("my-branch", func() error {
				called = true
				return nil
			})).To(Succeed())

			Expect(called).To(BeTrue())
			Expect(fakeGitClient.CheckoutCallCount()).To(Equal(1))
			Expect(fakeGitClient.CheckoutArgsForCall(0)).To(Equal("main"))
			Expect(fakeGitClient.CreateBranchCallCount()).To(Equal(1))
			Expect(fakeGitClient.CreateBranchArgsForCall(0)).To(Equal("my-branch"))
			Expect(fakeGitClient.RemoveAllCallCount()).To(Equal(1))
			Expect(fakeGitClient.AddCallCount()).To(Equal(1))
			Expect(fakeGitClient.AddArgsForCall(0)).To(Equal("."))
			Expect(fakeGitClient.CommitCallCount()).To(Equal(1))
		})

		When("Checkout fails", func() {
			BeforeEach(func() {
				fakeGitClient.CheckoutReturns(fmt.Errorf("failed"))
			})

			It("returns an error", func() {
				Expect(manager.CreateBranchWithContentFromMain("my-branch", func() error {
					return nil
				})).To(MatchError("failed to checkout main: failed"))
			})
		})

		When("CreateBranch fails", func() {
			BeforeEach(func() {
				fakeGitClient.CreateBranchReturns(fmt.Errorf("failed"))
			})

			It("returns an error", func() {
				Expect(manager.CreateBranchWithContentFromMain("my-branch", func() error {
					return nil
				})).To(MatchError("failed to create new branch my-branch: failed"))
			})
		})

		When("RemoveAll fails", func() {
			BeforeEach(func() {
				fakeGitClient.RemoveAllReturns(fmt.Errorf("failed"))
			})

			It("returns an error", func() {
				Expect(manager.CreateBranchWithContentFromMain("my-branch", func() error {
					return nil
				})).To(MatchError("failed to remove all: failed"))
			})
		})

		When("write content fails", func() {
			It("returns an error", func() {
				Expect(manager.CreateBranchWithContentFromMain("my-branch", func() error {
					return fmt.Errorf("failed")
				})).To(MatchError("failed to write content to repo: failed"))
			})
		})

		When("add fails", func() {
			BeforeEach(func() {
				fakeGitClient.AddReturns(fmt.Errorf("failed"))
			})

			It("returns an error", func() {
				Expect(manager.CreateBranchWithContentFromMain("my-branch", func() error {
					return nil
				})).To(MatchError("failed to add: failed"))
			})
		})

		When("Commit fails", func() {
			BeforeEach(func() {
				fakeGitClient.CommitReturns(fmt.Errorf("failed"))
			})

			It("returns an error", func() {
				Expect(manager.CreateBranchWithContentFromMain("my-branch", func() error {
					return nil
				})).To(MatchError("failed to commit: failed"))
			})
		})
	})

	Describe("MergeBranch", func() {
		BeforeEach(func() {
			fakeGitClient.MergeReturns([]string{"foo/bar"}, nil)
		})
		It("merges the branch and returns any conflicts", func() {
			mergeConflict, err := manager.MergeBranches("foo", "bar")
			Expect(err).NotTo(HaveOccurred())
			Expect(mergeConflict).To(ConsistOf("foo/bar"))
			Expect(fakeGitClient.CheckoutCallCount()).To(Equal(1))
			Expect(fakeGitClient.CheckoutArgsForCall(0)).To(Equal("foo"))
			Expect(fakeGitClient.MergeCallCount()).To(Equal(1))
			Expect(fakeGitClient.MergeArgsForCall(0)).To(Equal("bar"))
		})

		When("Checkout fails", func() {
			BeforeEach(func() {
				fakeGitClient.CheckoutReturns(fmt.Errorf("failed"))
			})

			It("returns an error", func() {
				_, err := manager.MergeBranches("foo", "bar")
				Expect(err).To(MatchError("failed to checkout main: failed"))
			})
		})

		When("Merge fails", func() {
			BeforeEach(func() {
				fakeGitClient.MergeReturns(nil, fmt.Errorf("failed"))
			})

			It("returns an error", func() {
				_, err := manager.MergeBranches("foo", "bar")
				Expect(err).To(MatchError("failed to merge branch foo into branch bar: failed"))
			})
		})
	})
})
