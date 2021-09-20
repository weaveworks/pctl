package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	"gopkg.in/yaml.v2"

	"github.com/weaveworks/pctl/pkg/log"
	"github.com/weaveworks/pctl/pkg/runner"
)

//Config contains the pctl bootstrap configuration
type Config struct {
	// GitRepository is the git repository flux resource the installation uses
	GitRepository profilesv1.GitRepository `yaml:"gitRepository,omitempty"`
	// DefaultDir defines the location to use with pctl add
	DefaultDir string `yaml:"defaultDir,omitempty"`
}

var r runner.Runner = &runner.CLIRunner{}

//CreateConfig creates the bootstrap config
func CreateConfig(cfg Config, directory string) error {
	gitDir, err := getGitRepoPath(directory)
	if err != nil {
		return err
	}

	pctlDir := filepath.Join(gitDir, ".pctl")
	if err := os.Mkdir(pctlDir, 0755); err != nil {
		return fmt.Errorf("failed to create .pctl dir %q: %w", pctlDir, err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(filepath.Join(pctlDir, "config.yaml"), data, 0644)
}

//GetConfig gets the bootstrap config
func GetConfig(directory string) *Config {
	gitDir, err := getGitRepoPath(directory)
	if err != nil {
		log.Warningf("failed to get git repo path: %v", err)
		return nil
	}
	configPath := filepath.Join(gitDir, ".pctl", "config.yaml")

	out, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		// don't use Warningf in case the file doesn't exist. Warning is a bit intrusive
		// the config file not existing is a perfectly fine scenario.
		fmt.Println("config file cannot be found... using default values")
		return nil
	} else if err != nil {
		log.Warningf("failed to read config file: %v", err)
		return nil
	}

	config := &Config{}
	if err = yaml.Unmarshal(out, config); err != nil {
		log.Warningf("failed to unmarshal config file: %v", err)
		return nil
	}
	return config
}

func getGitRepoPath(directory string) (string, error) {
	out, err := r.Run("git", "-C", directory, "rev-parse", "--show-toplevel")
	if err != nil {
		if strings.Contains(string(out), "not a git repository") {
			return "", fmt.Errorf("the target directory %q is not a git repository", directory)
		}
		return "", fmt.Errorf("failed to get git directory location: %w", err)
	}
	gitDir := strings.TrimSuffix(string(out), "\n")
	return gitDir, nil
}
