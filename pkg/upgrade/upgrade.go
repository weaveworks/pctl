package upgrade

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	copypkg "github.com/otiai10/copy"
	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/profile"
	"github.com/weaveworks/pctl/pkg/runner"
	"github.com/weaveworks/pctl/pkg/upgrade/branch"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	"sigs.k8s.io/yaml"
)

type UpgradeConfig struct {
	ProfileDir       string
	Version          string
	CatalogClient    catalog.CatalogClient
	CatalogManager   catalog.CatalogManager
	BranchManager    branch.BranchManager
	GitRepoName      string
	GitRepoNamespace string
	WorkingDir       string
}

var copy func(src, dest string) error = func(src, dest string) error {
	return copypkg.Copy(src, dest)
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

	//check new version exists
	_, err = cfg.CatalogManager.Show(cfg.CatalogClient, catalogName, profileName, cfg.Version)
	if err != nil {
		return fmt.Errorf("failed to get profile %q in catalog %q version %q: %w", profileName, catalogName, cfg.Version, err)
	}

	err = cfg.BranchManager.CreateRepoWithContent(func() error {
		installConfig := catalog.InstallConfig{
			Clients: catalog.Clients{
				CatalogClient: cfg.CatalogClient,
				ArtifactsMaker: profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					ProfileName:      profileName,
					GitClient:        git.NewCLIGit(git.CLIGitConfig{}, &runner.CLIRunner{}),
					RootDir:          cfg.WorkingDir,
					GitRepoNamespace: cfg.GitRepoNamespace,
					GitRepoName:      cfg.GitRepoName,
				}),
			},
			ProfileConfig: catalog.ProfileConfig{
				ProfileName: profileName,
				CatalogName: catalogName,
				Version:     currentVersion,
				ConfigMap:   profileInstallation.Spec.ConfigMap,
			},
		}
		if err := cfg.CatalogManager.Install(installConfig); err != nil {
			return fmt.Errorf("failed to install base profile: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create repository for upgrade: %w", err)
	}

	err = cfg.BranchManager.CreateBranchWithContentFromBase("user-changes", func() error {
		if err := copy(cfg.ProfileDir, cfg.WorkingDir); err != nil {
			return fmt.Errorf("failed to copy profile during upgrade: %w", err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create branch with user changes: %w", err)
	}

	err = cfg.BranchManager.CreateBranchWithContentFromBase("update-changes", func() error {
		installConfig := catalog.InstallConfig{
			Clients: catalog.Clients{
				CatalogClient: cfg.CatalogClient,
				ArtifactsMaker: profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					ProfileName:      profileName,
					GitClient:        git.NewCLIGit(git.CLIGitConfig{}, &runner.CLIRunner{}),
					RootDir:          cfg.WorkingDir,
					GitRepoNamespace: cfg.GitRepoNamespace,
					GitRepoName:      cfg.GitRepoName,
				}),
			},
			ProfileConfig: catalog.ProfileConfig{
				ProfileName: profileName,
				CatalogName: catalogName,
				Version:     cfg.Version,
				ConfigMap:   profileInstallation.Spec.ConfigMap,
			},
		}

		if err := cfg.CatalogManager.Install(installConfig); err != nil {
			return fmt.Errorf("failed to install update profile: %w", err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create branch with update changes: %w", err)
	}

	mergeConflict, err := cfg.BranchManager.MergeBranches("update-changes", "user-changes")
	if err != nil {
		return fmt.Errorf("failed to merge updates with user changes: %w", err)
	}
	if mergeConflict {
		fmt.Println("merge conflict")
	}

	if err := os.RemoveAll(cfg.ProfileDir); err != nil {
		return fmt.Errorf("failed to remove existing profile installation: %w", err)
	}

	if err := os.RemoveAll(filepath.Join(cfg.WorkingDir, ".git/")); err != nil {
		return fmt.Errorf("failed to remove git directory from upgrade directory: %w", err)
	}

	if err := copy(cfg.WorkingDir, cfg.ProfileDir); err != nil {
		return fmt.Errorf("failed to copy upgraded installation into installation directory: %w", err)
	}

	return nil
}
