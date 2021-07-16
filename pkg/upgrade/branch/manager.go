package branch

import (
	"fmt"

	"github.com/weaveworks/pctl/pkg/git"
)

//go:generate counterfeiter -o fakes/fake_branch_manager.go . BranchManager
type BranchManager interface {
	CreateRepoWithContent(contentWriter func() error) error
	CreateBranchWithContentFromBase(branch string, contentWriter func() error) error
	MergeBranches(branch1, branch2 string) (bool, error)
}

type Manager struct {
	workingDir string
	gitClient  git.Git
}

func NewManager(gitClient git.Git) BranchManager {
	return &Manager{
		workingDir: gitClient.GetDirectory(),
		gitClient:  gitClient,
	}
}

func (m *Manager) CreateRepoWithContent(writeContent func() error) error {
	if err := m.gitClient.Init(); err != nil {
		return fmt.Errorf("failed to init git repo: %w", err)
	}

	if err := writeContent(); err != nil {
		return fmt.Errorf("failed to copy profile during upgrade: %w", err)
	}

	if err := m.gitClient.Add(); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := m.gitClient.Commit(); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}
	return nil
}

func (m *Manager) CreateBranchWithContentFromBase(branch string, writeContent func() error) error {
	if err := m.gitClient.Checkout("main"); err != nil {
		return fmt.Errorf("failed to checkout main: %w", err)
	}

	if err := m.gitClient.CreateNewBranch(branch); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := m.gitClient.RemoveAll(); err != nil {
		return fmt.Errorf("failed to remove content: %w", err)
	}

	if err := writeContent(); err != nil {
		return fmt.Errorf("failed to copy profile during upgrade: %w", err)
	}

	if err := m.gitClient.Add(); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := m.gitClient.Commit(); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}
	return nil
}

func (m *Manager) MergeBranches(branch1, branch2 string) (bool, error) {
	if err := m.gitClient.Checkout(branch1); err != nil {
		return false, fmt.Errorf("failed to checkout main: %w", err)
	}

	mergeConflict, err := m.gitClient.Merge("user-changes")
	if err != nil {
		return false, fmt.Errorf("failed to add: %w", err)
	}

	if mergeConflict {
		fmt.Println("merge conflict")
	}
	return false, nil
}
