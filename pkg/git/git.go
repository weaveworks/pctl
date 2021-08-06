package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	CreateBranch(string) error
	// IsRepository returns whether a location is a git repository or not.
	IsRepository() error
	// HasChanges returns whether a location has uncommitted changes or not.
	HasChanges() (bool, error)
	// Push will push to a remote.
	Push() error
	// Clone will take a repo location and clone it into a given folder.
	Clone(repo, branch, location string) error
	// Init will create a git repository
	Init() error
	// Merge will merge the branch into the currently checked out branch
	// return a list of files when a conflict occurs
	Merge(branch string) ([]string, error)
	// Checkout the target branch
	Checkout(branch string) error
	// GetDirectory returns the git directory
	GetDirectory() string
	// RemoveAll files from git repository
	RemoveAll() error
}

// CLIGitConfig defines configuration options for CLIGit.
type CLIGitConfig struct {
	Directory string
	Branch    string
	Remote    string
	Message   string
	Base      string
	Quiet     bool
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

func (g *CLIGit) RemoveAll() error {
	if err := g.Add(); err != nil {
		return err
	}

	args := []string{
		"--git-dir", filepath.Join(g.Directory, ".git"),
		"--work-tree", g.Directory,
		"rm",
		"-rf",
		".",
	}
	if err := g.runGitCmd(args...); err != nil {
		return fmt.Errorf("failed to run rm: %w\n", err)
	}
	return nil
}
func (g *CLIGit) GetDirectory() string {
	return g.Directory
}

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
		return fmt.Errorf("failed to run clone: %w\n", err)
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
		g.Message,
	}
	if err := g.runGitCmd(args...); err != nil {
		return fmt.Errorf("failed to run commit: %w", err)
	}
	return nil
}

// CreateBranch creates a branch if it differs from the base.
func (g *CLIGit) CreateBranch(branch string) error {
	if branch == g.Base {
		return nil
	}
	g.printf("creating new branch\n")
	args := []string{
		"--git-dir", filepath.Join(g.Directory, ".git"),
		"--work-tree", g.Directory,
		"checkout",
		"-b",
		branch,
	}
	if err := g.runGitCmd(args...); err != nil {
		return fmt.Errorf("failed to create new branch %s: %w", branch, err)
	}
	return nil
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
	g.printf("pushing to remote\n")
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
	g.printf("adding unstaged changes\n")
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
		g.printf("failed to run git with output: %s\n", string(out))
	}
	return err
}

// Init will initalise the git repository
func (g *CLIGit) Init() error {
	args := []string{
		"init",
		g.Directory,
		"-b",
		"main",
	}
	if err := g.runGitCmd(args...); err != nil {
		return fmt.Errorf("failed to init: %w", err)
	}
	return nil
}

// Merge will merge the branch into the currently checked out branch.
// returns (true, nil) when merge conflict occurs.
func (g *CLIGit) Merge(branch string) ([]string, error) {
	args := []string{
		"--git-dir", filepath.Join(g.Directory, ".git"),
		"--work-tree", g.Directory,
		"merge",
		branch,
	}

	out, err := g.Runner.Run("git", args...)
	if err != nil {
		if strings.Contains(string(out), "Merge conflict") {
			args := []string{
				"--git-dir", filepath.Join(g.Directory, ".git"),
				"--work-tree", g.Directory,
				"diff",
				"--name-only",
				"--diff-filter=U",
			}

			outBytes, err := g.Runner.Run("git", args...)
			if err != nil {
				return nil, fmt.Errorf("failed to list files with merge conflicts: %w", err)
			}
			out := strings.TrimSuffix(string(outBytes), "\n")
			return strings.Split(string(out), "\n"), nil
		}
		return nil, fmt.Errorf("failed to run merge: %w", err)
	}
	return nil, nil
}

// Checkout the target branch
func (g *CLIGit) Checkout(branch string) error {
	args := []string{
		"--git-dir", filepath.Join(g.Directory, ".git"),
		"--work-tree", g.Directory,
		"checkout",
		branch,
	}

	if err := g.runGitCmd(args...); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branch, err)
	}
	return nil
}

func (g *CLIGit) printf(format string, a ...interface{}) {
	if !g.Quiet {
		fmt.Printf(format, a...)
	}
}
