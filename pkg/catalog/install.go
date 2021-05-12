package catalog

import (
	"fmt"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/writer"
)

// InstallConfig defines parameters for the installation call.
type InstallConfig struct {
	CatalogClient CatalogClient
	Branch        string
	CatalogName   string
	ConfigMap     string
	Namespace     string
	ProfileName   string
	SubName       string
	Writer        writer.Writer
	Version       string
}

// Install using the catalog at catalogURL and a profile matching the provided profileName generates a profile subscription
// writing it out with the provided profile subscription writer.
func Install(cfg InstallConfig) error {
	profile, err := Show(cfg.CatalogClient, WithCatalogName(cfg.CatalogName), WithProfileName(cfg.ProfileName), WithCatalogVersion(cfg.Version))
	if err != nil {
		return fmt.Errorf("failed to get profile %q in catalog %q: %w", cfg.ProfileName, cfg.CatalogName, err)
	}

	subscription := profilesv1.ProfileSubscription{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ProfileSubscription",
			APIVersion: "weave.works/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.SubName,
			Namespace: cfg.Namespace,
		},
		Spec: profilesv1.ProfileSubscriptionSpec{
			ProfileURL: profile.URL,
			Branch:     cfg.Branch,
		},
	}
	if cfg.ConfigMap != "" {
		subscription.Spec.ValuesFrom = []helmv2.ValuesReference{
			{
				Kind:      "ConfigMap",
				Name:      cfg.SubName + "-values",
				ValuesKey: cfg.ConfigMap,
			},
		}
	}
	if err := cfg.Writer.Output(&subscription); err != nil {
		return fmt.Errorf("failed to output subscription information: %w", err)
	}
	return nil
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
