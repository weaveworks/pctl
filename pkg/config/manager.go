package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/weaveworks/pctl/pkg/runner"
)

func Create(gitRepository string) error {
	r := runner.CLIRunner{}
	out, err := r.Run("git", "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("failed to get git dir location: %w", err)
	}
	pctlDir := filepath.Join(strings.TrimSuffix(string(out), "\n"), ".pctl")
	if err := os.Mkdir(pctlDir, 0755); err != nil {
		return fmt.Errorf("failed to create .pctl dir %q: %w", pctlDir, err)
	}
	content := []byte(gitRepository)
	return os.WriteFile(filepath.Join(pctlDir, "config"), content, 0644)
}

func Get() (string, error) {
	r := runner.CLIRunner{}
	out, err := r.Run("git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("failed to get git dir location: %w", err)
	}
	configPath := filepath.Join(strings.TrimSuffix(string(out), "\n"), ".pctl", "config")

	out, err = os.ReadFile(configPath)
	return string(out), err
}
