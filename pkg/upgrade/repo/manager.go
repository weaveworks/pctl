package repo

import (
	"fmt"

	"github.com/weaveworks/pctl/pkg/git"
)

//go:generate counterfeiter -o fakes/fake_repo_manager.go . RepoManager
type RepoManager interface {
	CreateRepoWithContent(contentWriter func() error) error
	CreateBranchWithContentFromMain(branch string, contentWriter func() error) error
	MergeBranches(branch1, branch2 string) (bool, error)
}

type Manager struct {
	workingDir string
	gitClient  git.Git
}

func NewManager(gitClient git.Git) RepoManager {
	return &Manager{
		workingDir: gitClient.GetDirectory(),
		gitClient:  gitClient,
	}
}

func (m *Manager) CreateRepoWithContent(writeContent func() error) error {
	if err := m.gitClient.Init(); err != nil {
		return fmt.Errorf("failed to init repo: %w", err)
	}

	if err := writeContent(); err != nil {
		return fmt.Errorf("failed to write content to repo: %w", err)
	}

	if err := m.gitClient.Add(); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := m.gitClient.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return nil
}

func (m *Manager) CreateBranchWithContentFromMain(branch string, writeContent func() error) error {
	if err := m.gitClient.Checkout("main"); err != nil {
		return fmt.Errorf("failed to checkout main: %w", err)
	}

	if err := m.gitClient.CreateNewBranch(branch); err != nil {
		return fmt.Errorf("failed to create new branch %s: %w", branch, err)
	}

	if err := m.gitClient.RemoveAll(); err != nil {
		return fmt.Errorf("failed to remove all: %w", err)
	}

	if err := writeContent(); err != nil {
		return fmt.Errorf("failed to write content to repo: %w", err)
	}

	if err := m.gitClient.Add(); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := m.gitClient.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return nil
}

func (m *Manager) MergeBranches(branch1, branch2 string) (bool, error) {
	if err := m.gitClient.Checkout(branch1); err != nil {
		return false, fmt.Errorf("failed to checkout main: %w", err)
	}

	mergeConflict, err := m.gitClient.Merge(branch2)
	if err != nil {
		return false, fmt.Errorf("failed to merge branch %s into branch %s: %w", branch1, branch2, err)
	}

	return mergeConflict, nil
}