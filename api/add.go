package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/weaveworks/pctl/pkg/bootstrap"
	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/client"
	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/install"
	"github.com/weaveworks/pctl/pkg/log"
	"github.com/weaveworks/pctl/pkg/runner"
)

// AddProfileOpts defines settings for adding profiles.
type AddProfileOpts struct {
	URL           string
	Branch        string
	SubName       string
	Namespace     string
	ConfigMap     string
	Dir           string
	Path          string
	Message       string
	GitRepository string
	ProfilePath   string
	CatalogClient *client.Client
}

// AddProfile add runs the add part of the `add` command.
func AddProfile(opts AddProfileOpts) (string, error) {
	var (
		catalogName, version, profileName string
	)
	if opts.URL == "" {
		parts := strings.Split(opts.ProfilePath, "/")
		if len(parts) < 2 {
			return "", errors.New("both catalog name and profile name must be provided")
		}
		if len(parts) == 3 {
			version = parts[2]
		}
		catalogName, profileName = parts[0], parts[1]
	}

	var source string
	if opts.URL != "" && opts.Path != "" {
		source = fmt.Sprintf("repository %s, path: %s and branch %s", opts.URL, opts.Path, opts.Branch)
	} else if opts.URL != "" && opts.Path == "" {
		source = fmt.Sprintf("repository %s and branch %s", opts.URL, opts.Branch)
	} else {
		source = fmt.Sprintf("catalog entry %s/%s/%s", catalogName, profileName, version)
	}

	log.Actionf("generating profile installation from source: %s", source)
	r := &runner.CLIRunner{}
	g := git.NewCLIGit(git.CLIGitConfig{
		Message: opts.Message,
	}, r)

	gitRepoNamespace, gitRepoName, err := getGitRepositoryNamespaceAndName(opts.GitRepository)
	if err != nil {
		return "", err
	}

	installationDirectory := filepath.Join(opts.Dir, opts.SubName)
	installer := install.NewInstaller(install.Config{
		GitClient:        g,
		RootDir:          installationDirectory,
		GitRepoNamespace: gitRepoNamespace,
		GitRepoName:      gitRepoName,
	})
	cfg := catalog.InstallConfig{
		Clients: catalog.Clients{
			CatalogClient: opts.CatalogClient,
			Installer:     installer,
		},
		Profile: catalog.Profile{
			ProfileConfig: catalog.ProfileConfig{
				CatalogName:   catalogName,
				ConfigMap:     opts.ConfigMap,
				Namespace:     opts.Namespace,
				Path:          opts.Path,
				ProfileBranch: opts.Branch,
				ProfileName:   profileName,
				SubName:       opts.SubName,
				URL:           opts.URL,
				Version:       version,
			},
			GitRepoConfig: catalog.GitRepoConfig{
				Namespace: gitRepoNamespace,
				Name:      gitRepoName,
			},
		},
	}
	manager := &catalog.Manager{}
	return installationDirectory, manager.Install(cfg)
}

func getGitRepositoryNamespaceAndName(gitRepository string) (string, string, error) {
	if gitRepository != "" {
		split := strings.Split(gitRepository, "/")
		if len(split) != 2 {
			return "", "", fmt.Errorf("git-repository must in format <namespace>/<name>; was: %s", gitRepository)
		}
		return split[0], split[1], nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch current working directory: %w", err)
	}
	config, err := bootstrap.GetConfig(wd)
	if err == nil && config != nil {
		return config.GitRepository.Namespace, config.GitRepository.Name, nil
	}
	return "", "", fmt.Errorf("flux git repository not provided, please provide the --git-repository flag or use the pctl bootstrap functionality")
}
