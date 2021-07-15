package branch

import (
	"fmt"
	"os"

	"github.com/weaveworks/pctl/pkg/git"
)

type Manager struct {
	WorkingDir string
	GitClient  git.Git
}

func (m *Manager) CreateRepoWithBaseBranch(writeContent func() error) error {
	if err := os.Mkdir(m.WorkingDir, 0755); err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}

	if err := writeContent(); err != nil {
		return fmt.Errorf("failed to copy profile during upgrade: %w", err)
	}

	if err := m.GitClient.Add(); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := m.GitClient.Commit(); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}
	return nil
}

func (m *Manager) CreateBranchWithContent(branch string, writeContent func() error) error {
	if err := m.GitClient.Checkout("main"); err != nil {
		return fmt.Errorf("failed to checkout main: %w", err)
	}

	if err := m.GitClient.CreateNewBranch(branch); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := os.RemoveAll(m.WorkingDir); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := os.Mkdir(m.WorkingDir, 0755); err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}

	if err := writeContent(); err != nil {
		return fmt.Errorf("failed to copy profile during upgrade: %w", err)
	}

	if err := m.GitClient.Add(); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := m.GitClient.Commit(); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}
	return nil
}
