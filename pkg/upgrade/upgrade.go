package upgrade

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/otiai10/copy"
	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/profile"
	"github.com/weaveworks/pctl/pkg/runner"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	"sigs.k8s.io/yaml"
)

type UpgradeConfig struct {
	ProfileDir       string
	Version          string
	CatalogClient    catalog.CatalogClient
	CatalogManager   catalog.CatalogManager
	GitClient        git.Git
	GitRepoName      string
	GitRepoNamespace string
}

func Upgrade(cfg UpgradeConfig) error {
	out, err := ioutil.ReadFile(path.Join(cfg.ProfileDir, "profile-installation.yaml"))
	if err != nil {
		return fmt.Errorf("failed to read profile installation: %w", err)
	}

	var profileInstallation profilesv1.ProfileInstallation
	if err := yaml.Unmarshal(out, &profileInstallation); err != nil {
		return fmt.Errorf("failed to parse profile installation: %w", err)
	}

	catalogName := profileInstallation.Spec.Catalog.Catalog
	profileName := profileInstallation.Spec.Catalog.Profile
	currentVersion := profileInstallation.Spec.Catalog.Version
	_, err = cfg.CatalogManager.Show(cfg.CatalogClient, catalogName, profileName, currentVersion)
	if err != nil {
		return fmt.Errorf("failed to get profile %q in catalog %q version %q: %w", profileName, catalogName, currentVersion, err)
	}

	_, err = cfg.CatalogManager.Show(cfg.CatalogClient, catalogName, profileName, cfg.Version)
	if err != nil {
		return fmt.Errorf("failed to get profile %q in catalog %q version %q: %w", profileName, catalogName, cfg.Version, err)
	}

	if err := cfg.GitClient.Init(); err != nil {
		return fmt.Errorf("failed to init git repo: %w", err)
	}
	//use a working directory inside the git directory, to prevent copying .git directory
	workingDir := filepath.Join(cfg.GitClient.GetDirectory(), "content")

	installConfig := catalog.InstallConfig{
		Clients: catalog.Clients{
			CatalogClient: cfg.CatalogClient,
			ArtifactsMaker: profile.NewProfilesArtifactsMaker(profile.MakerConfig{
				ProfileName:      profileName,
				GitClient:        git.NewCLIGit(git.CLIGitConfig{}, &runner.CLIRunner{}),
				RootDir:          workingDir,
				GitRepoNamespace: cfg.GitRepoNamespace,
				GitRepoName:      cfg.GitRepoName,
			}),
		},
		ProfileConfig: catalog.ProfileConfig{
			ProfileName: profileName,
			CatalogName: catalogName,
			Version:     currentVersion,
		},
	}

	if err := createBaseBranch(cfg.GitClient, workingDir, installConfig, cfg.CatalogManager); err != nil {
		return err
	}

	if err := createBranchWithUserChanges(cfg.GitClient, workingDir, cfg.ProfileDir); err != nil {
		return err
	}

	installConfig = catalog.InstallConfig{
		Clients: catalog.Clients{
			CatalogClient: cfg.CatalogClient,
			ArtifactsMaker: profile.NewProfilesArtifactsMaker(profile.MakerConfig{
				ProfileName:      profileName,
				GitClient:        git.NewCLIGit(git.CLIGitConfig{}, &runner.CLIRunner{}),
				RootDir:          workingDir,
				GitRepoNamespace: cfg.GitRepoNamespace,
				GitRepoName:      cfg.GitRepoName,
			}),
		},
		ProfileConfig: catalog.ProfileConfig{
			ProfileName: profileName,
			CatalogName: catalogName,
			Version:     cfg.Version,
		},
	}
	if err := createBranchWithNewVersion(cfg.GitClient, workingDir, installConfig, cfg.CatalogManager); err != nil {
		return err
	}

	mergeConflict, err := cfg.GitClient.Merge("user-changes")
	if err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}
	if mergeConflict {
		fmt.Println("merge conflict")
	}

	return nil
}

func createBaseBranch(gitClient git.Git, workingDir string, installConfig catalog.InstallConfig, catalogManager catalog.CatalogManager) error {
	if err := os.Mkdir(workingDir, 0755); err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}

	if err := catalogManager.Install(installConfig); err != nil {
		return fmt.Errorf("failed to install base profile: %w", err)
	}

	if err := gitClient.Add(); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := gitClient.Commit(); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}
	return nil
}

func createBranchWithUserChanges(gitClient git.Git, workingDir, profileDir string) error {
	if err := gitClient.CreateNewBranch("user-changes"); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := os.RemoveAll(workingDir); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := os.Mkdir(workingDir, 0755); err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}

	if err := copy.Copy(profileDir, workingDir); err != nil {
		return fmt.Errorf("failed to copy profile during upgrade: %w", err)
	}

	if err := gitClient.Add(); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := gitClient.Commit(); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}
	return nil
}

func createBranchWithNewVersion(gitClient git.Git, workingDir string, installConfig catalog.InstallConfig, catalogManager catalog.CatalogManager) error {
	if err := gitClient.Checkout("main"); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := gitClient.CreateNewBranch("update-changes"); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := os.RemoveAll(workingDir); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := os.Mkdir(workingDir, 0755); err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}

	if err := catalogManager.Install(installConfig); err != nil {
		return fmt.Errorf("failed to install update profile: %w", err)
	}

	if err := gitClient.Add(); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}

	if err := gitClient.Commit(); err != nil {
		return fmt.Errorf("failed to add: %w", err)
	}
	return nil
}
