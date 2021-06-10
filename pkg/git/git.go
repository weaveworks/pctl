package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/weaveworks/pctl/pkg/runner"
)

const (
	gitCmd = "git"
)

// Git defines high level abilities for Git related operations.
//go:generate counterfeiter -o fakes/fake_git.go . Git
type Git interface {
	// Add staged changes.
	Add() error
	// Commit changes.
	Commit() error
	// CreateBranch create a branch if it's needed.
	CreateBranch() error
	// CreateRepository bootstraps a plain repository at a given location.
	CreateRepository() error
	// IsRepository returns whether a location is a git repository or not.
	IsRepository() error
	// HasChanges returns whether a location has uncommitted changes or not.
	HasChanges() (bool, error)
	// Push will push to a remote.
	Push() error
	// Clone will take a repo location and clone it into a given folder.
	Clone(repo, branch, location string) error
	// SparseClone will take a repo location and clone it into a given folder.
	SparseClone(repo, branch, location, path string) error
}

// CLIGitConfig defines configuration options for CLIGit.
type CLIGitConfig struct {
	Directory string
	Branch    string
	Remote    string
	Base      string
}

// CLIGit is a new command line based Git.
type CLIGit struct {
	CLIGitConfig
	Runner runner.Runner
}

// NewCLIGit creates a new command line based Git.
func NewCLIGit(cfg CLIGitConfig, r runner.Runner) *CLIGit {
	return &CLIGit{
		CLIGitConfig: cfg,
		Runner:       r,
	}
}

// Make sure CLIGit implements all the required methods.
var _ Git = &CLIGit{}

// Clone will take a repo location and clone it into a given folder.
func (g *CLIGit) Clone(repo, branch, location string) error {
	args := []string{
		"clone",
		"--branch",
		branch,
		"--depth",
		"1",
		repo,
		location,
	}
	if err := g.runGitCmd(args...); err != nil {
		return fmt.Errorf("failed to run clone: %w", err)
	}
	return nil
}

// Commit all changes.
func (g *CLIGit) Commit() error {
	hasChanges, err := g.HasChanges()
	if err != nil {
		return fmt.Errorf("failed to detect if repository has changes: %w", err)
	}
	if !hasChanges {
		return nil
	}
	args := []string{
		"--git-dir", filepath.Join(g.Directory, ".git"),
		"--work-tree", g.Directory,
		"commit",
		"-am",
		"Push changes to remote",
	}
	if err := g.runGitCmd(args...); err != nil {
		return fmt.Errorf("failed to run commit: %w", err)
	}
	return nil
}

// SparseClone will take a repo location and clone it into a given folder using sparse clone.
// This might take a bit longer initially, but is much more performant in case of a mono profiles repository.
func (g *CLIGit) SparseClone(repo, branch, location, path string) error {
	// in case the profile data is in root, we have a single profile / repository.
	// clone everything.
	if path == "." || path == "" {
		return g.Clone(repo, branch, location)
	}
	// otherwise, we do a sparse checkout to optimize for a single folder.
	args := []string{
		"clone",
		"--branch",
		branch,
		"--depth",
		"1",
		"--sparse",
		"--filter",
		"blob:none",
		repo,
		location,
	}
	if err := g.runGitCmd(args...); err != nil {
		return fmt.Errorf("failed to run clone: %w", err)
	}
	args = []string{
		"--git-dir", filepath.Join(location, ".git"),
		"--work-tree", location,
		"sparse-checkout",
		"set",
		path,
	}
	if err := g.runGitCmd(args...); err != nil {
		return fmt.Errorf("failed to run sparse checkout: %w", err)
	}
	return nil
}

// CreateBranch creates a branch if it differs from the base.
func (g *CLIGit) CreateBranch() error {
	if g.Base == g.Branch {
		return nil
	}
	fmt.Println("creating new branch")
	args := []string{
		"--git-dir", filepath.Join(g.Directory, ".git"),
		"--work-tree", g.Directory,
		"checkout",
		"-b",
		g.Branch,
	}
	if err := g.runGitCmd(args...); err != nil {
		return fmt.Errorf("failed to create new branch %s: %w", g.Branch, err)
	}
	return nil
}

// CreateRepository bootstraps a plain repository at a given location.
func (g *CLIGit) CreateRepository() error {
	return errors.New("implement me")
}

// IsRepository returns whether a location is a git repository or not.
func (g *CLIGit) IsRepository() error {
	// Note that this is redundant in case of CLI git, because the git command line utility
	// already checks if the given location is a repository or not. Never the less we do this
	// for posterity.
	if _, err := os.Stat(filepath.Join(g.Directory, ".git")); err != nil {
		return err
	}
	return nil
}

// HasChanges returns whether a location has uncommitted changes or not.
func (g *CLIGit) HasChanges() (bool, error) {
	args := []string{
		"--git-dir", filepath.Join(g.Directory, ".git"),
		"--work-tree", g.Directory,
		"status",
		"-s",
	}
	out, err := g.Runner.Run(gitCmd, args...)
	if err != nil {
		return false, fmt.Errorf("failed to check if there are changes: %w", err)
	}
	return string(out) != "", nil
}

// Push will push to a remote.
func (g *CLIGit) Push() error {
	fmt.Println("pushing to remote")
	args := []string{
		"--git-dir", filepath.Join(g.Directory, ".git"),
		"--work-tree", g.Directory,
		"push",
		g.Remote,
		g.Branch,
	}
	if err := g.runGitCmd(args...); err != nil {
		return fmt.Errorf("failed to push changes to remote %s with branch %s: %w", g.Remote, g.Branch, err)
	}
	return nil
}

// Add will add any changes to the generated file.
func (g *CLIGit) Add() error {
	fmt.Println("adding unstaged changes")
	args := []string{
		"--git-dir", filepath.Join(g.Directory, ".git"),
		"--work-tree", g.Directory,
		"add",
		".",
	}
	if err := g.runGitCmd(args...); err != nil {
		return fmt.Errorf("failed to run add: %w", err)
	}
	return nil
}

// runGitCmd is a convenient wrapper around running commands with error output when the output is not needed but needs to
// be logged.
func (g *CLIGit) runGitCmd(args ...string) error {
	out, err := g.Runner.Run(gitCmd, args...)
	if err != nil {
		fmt.Printf("failed to run git with output: %s\n", string(out))
	}
	return err
}
