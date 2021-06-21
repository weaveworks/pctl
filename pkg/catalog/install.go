package catalog

import (
	"fmt"
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/profile"
)

// Clients contains a set of clients which are used by install.
type Clients struct {
	CatalogClient  CatalogClient
	ArtifactsMaker profile.ArtifactsMaker
}

// ProfileConfig contains configuration for profiles ie. catalogName, profilesName, etc.
type ProfileConfig struct {
	CatalogName   string
	ConfigMap     string
	Namespace     string
	ProfileName   string
	SubName       string
	Version       string
	ProfileBranch string
	URL           string
	Path          string
}

// InstallConfig defines parameters for the installation call.
type InstallConfig struct {
	Clients
	ProfileConfig
}

// Install using the catalog at catalogURL and a profile matching the provided profileName generates a profile installation
// and its artifacts
func Install(cfg InstallConfig) error {
	pSpec, err := getProfileSpec(cfg)
	if err != nil {
		return err
	}
	installation := profilesv1.ProfileInstallation{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ProfileInstallation",
			APIVersion: "weave.works/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.SubName,
			Namespace: cfg.Namespace,
		},
		Spec: pSpec,
	}
	if cfg.ConfigMap != "" {
		installation.Spec.ValuesFrom = []helmv2.ValuesReference{
			{
				Kind:      "ConfigMap",
				Name:      cfg.SubName + "-values",
				ValuesKey: cfg.ConfigMap,
			},
		}
	}
	if err := cfg.ArtifactsMaker.Make(installation); err != nil {
		return fmt.Errorf("failed to make artifacts: %w", err)
	}
	return nil
}

// getProfileSpec generates a spec based on configured properties.
func getProfileSpec(cfg InstallConfig) (profilesv1.ProfileInstallationSpec, error) {
	if cfg.URL != "" {
		return profilesv1.ProfileInstallationSpec{
			Source: &profilesv1.Source{
				URL:    cfg.URL,
				Branch: cfg.ProfileBranch,
				Path:   cfg.Path,
			},
		}, nil
	}
	p, err := Show(cfg.CatalogClient, cfg.CatalogName, cfg.ProfileName, cfg.Version)
	if err != nil {
		return profilesv1.ProfileInstallationSpec{}, fmt.Errorf("failed to get profile %q in catalog %q: %w", cfg.ProfileName, cfg.CatalogName, err)
	}

	//tag could be <semver> or <name/semver>
	path := "."
	splitTag := strings.Split(p.Tag, "/")
	if len(splitTag) > 1 {
		path = splitTag[0]
	}

	return profilesv1.ProfileInstallationSpec{
		Source: &profilesv1.Source{
			URL:  p.URL,
			Tag:  p.Tag,
			Path: path,
		},
		Catalog: &profilesv1.Catalog{
			Catalog: cfg.CatalogName,
			Version: profilesv1.GetVersionFromTag(p.Tag),
			Profile: p.Name,
		},
	}, nil
}

// CreatePullRequest creates a pull request from the current changes.
func CreatePullRequest(scm git.SCMClient, g git.Git) error {
	if err := g.IsRepository(); err != nil {
		return fmt.Errorf("directory is not a git repository: %w", err)
	}

	if err := g.CreateBranch(); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	if err := g.Add(); err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	if err := g.Commit(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	if err := g.Push(); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	if err := scm.CreatePullRequest(); err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}
	return nil
}
