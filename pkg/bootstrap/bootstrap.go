package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/weaveworks/pctl/pkg/runner"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	"gopkg.in/yaml.v2"
)

//Config contains the pctl bootstrap configuration
type Config struct {
	// GitRepository is the git repository flux resource the installation uses
	GitRepository profilesv1.GitRepository `json:"gitRepository,omitempty"`
}

var r runner.Runner = &runner.CLIRunner{}

//CreateConfig creates the bootstrap config
func CreateConfig(namespace, name, directory string) error {
	out, err := r.Run("git", "-C", directory, "rev-parse", "--show-toplevel")
	if err != nil {
		if strings.Contains(string(out), "not a git repository") {
			return fmt.Errorf("the target directory %q is not a git repository", directory)
		}
		return fmt.Errorf("failed to get git directory location: %w", err)
	}

	pctlDir := filepath.Join(strings.TrimSuffix(string(out), "\n"), ".pctl")
	if err := os.Mkdir(pctlDir, 0755); err != nil {
		return fmt.Errorf("failed to create .pctl dir %q: %w", pctlDir, err)
	}

	data, err := yaml.Marshal(Config{
		GitRepository: profilesv1.GitRepository{
			Name:      name,
			Namespace: namespace,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(filepath.Join(pctlDir, "config.yaml"), data, 0644)
}

//GetConfig gets the bootstrap config
func GetConfig(directory string) (*Config, error) {
	out, err := r.Run("git", "-C", directory, "rev-parse", "--show-toplevel")
	if err != nil {
		if strings.Contains(string(out), "not a git repository") {
			return nil, fmt.Errorf("the target directory %q is not a git repository", directory)
		}
		return nil, fmt.Errorf("failed to get git directory location: %w", err)
	}
	configPath := filepath.Join(strings.TrimSuffix(string(out), "\n"), ".pctl", "config.yaml")

	out, err = os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{}
	if err = yaml.Unmarshal(out, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}
	return config, nil
}
