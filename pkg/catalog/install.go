package catalog

import (
	"fmt"
	"os"
	"path/filepath"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/profile"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
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
	Version       string
	Directory     string
}

//MakeArtifacts returns artifacts for a subscription
type MakeArtifacts func(sub profilesv1.ProfileSubscription) ([]profile.Artifact, error)

var makeArtifacts = profile.MakeArtifacts

// Install using the catalog at catalogURL and a profile matching the provided profileName generates a profile subscription
// and its artifacts
func Install(cfg InstallConfig) error {
	profile, err := Show(cfg.CatalogClient, cfg.CatalogName, cfg.ProfileName, cfg.Version)
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
			Version:    filepath.Join(profile.Name, profile.Version),
			ProfileCatalogDescription: &profilesv1.ProfileCatalogDescription{
				Catalog: cfg.CatalogName,
				Version: profile.Version,
				Profile: profile.Name,
			},
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

	artifacts, err := makeArtifacts(subscription)
	if err != nil {
		return fmt.Errorf("failed to generate artifacts: %w", err)
	}

	e := kjson.NewSerializerWithOptions(kjson.DefaultMetaFactory, nil, nil, kjson.SerializerOptions{Yaml: true, Strict: true})
	generateOutput := func(filename string, o runtime.Object) error {
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
		if err != nil {
			return err
		}
		defer func(f *os.File) {
			if err := f.Close(); err != nil {
				fmt.Printf("Failed to properly close file %s\n", f.Name())
			}
		}(f)
		if err := e.Encode(o, f); err != nil {
			return err
		}
		return nil
	}

	profileRootdir := filepath.Join(cfg.Directory, profile.Name)
	artifactsRootDir := filepath.Join(profileRootdir, "artifacts")

	for _, artifact := range artifacts {
		artifactDir := filepath.Join(artifactsRootDir, artifact.Name)
		if err = os.MkdirAll(artifactDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory")
		}
		for _, obj := range artifact.Objects {
			filename := filepath.Join(artifactDir, fmt.Sprintf("%s.%s", obj.GetObjectKind().GroupVersionKind().Kind, "yaml"))
			if err := generateOutput(filename, obj); err != nil {
				return err
			}
		}
	}

	return generateOutput(filepath.Join(profileRootdir, "profile.yaml"), &subscription)
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
