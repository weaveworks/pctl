package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

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
	URL           string
	Path          string
}

// domainRegex is used to verify branch
var domainRegex = regexp.MustCompile(`[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*`)

//MakeArtifacts returns artifacts for a subscription
type MakeArtifacts func(sub profilesv1.ProfileSubscription) ([]runtime.Object, error)

var makeArtifacts = profile.MakeArtifacts

// Install using the catalog at catalogURL and a profile matching the provided profileName generates a profile subscription
// and its artifacts
func Install(cfg InstallConfig) error {
	pSpec, err := getProfileSpec(cfg)
	if err != nil {
		return err
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
		Spec: pSpec,
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

	directory := filepath.Join(cfg.Directory, cfg.ProfileName)
	if cfg.ProfileName == "" {
		directory = filepath.Join(cfg.Directory, cfg.Path)
	}
	if err = os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory")
	}

	e := kjson.NewSerializerWithOptions(kjson.DefaultMetaFactory, nil, nil, kjson.SerializerOptions{Yaml: true, Strict: true})
	generateOutput := func(filename string, o runtime.Object) error {
		f, err := os.OpenFile(filepath.Join(directory, filename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
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

	for i, a := range artifacts {
		filename := fmt.Sprintf("%s-%d.%s", a.GetObjectKind().GroupVersionKind().Kind, i, "yaml")
		if err := generateOutput(filename, a); err != nil {
			return err
		}
	}

	return generateOutput("profile.yaml", &subscription)
}

// getProfileSpec generates a spec based on configured properties.
func getProfileSpec(cfg InstallConfig) (profilesv1.ProfileSubscriptionSpec, error) {
	if cfg.URL != "" {
		if cfg.Path == "" {
			return profilesv1.ProfileSubscriptionSpec{}, errors.New("path must be provided with url")
		}
		// The regex matches characters. It will not match characters which aren't allowed
		// that will result in multiple matched groups. If the whole thing isn't a match
		// that's determined by the fact that there are multiple groups. There should be only one.
		if m := domainRegex.FindAllStringSubmatch(cfg.Branch, -1); len(m) > 1 {
			return profilesv1.ProfileSubscriptionSpec{}, errors.New("branch must match RFC 1123 subdomain format")
		}
		return profilesv1.ProfileSubscriptionSpec{
			ProfileURL: cfg.URL,
			Branch:     cfg.Branch,
			Path:       cfg.Path,
		}, nil
	}
	p, err := Show(cfg.CatalogClient, cfg.CatalogName, cfg.ProfileName, cfg.Version)
	if err != nil {
		return profilesv1.ProfileSubscriptionSpec{}, fmt.Errorf("failed to get profile %q in catalog %q: %w", cfg.ProfileName, cfg.CatalogName, err)
	}

	return profilesv1.ProfileSubscriptionSpec{
		ProfileURL: p.URL,
		Version:    filepath.Join(p.Name, p.Version),
		ProfileCatalogDescription: &profilesv1.ProfileCatalogDescription{
			Catalog: cfg.CatalogName,
			Version: cfg.Version,
			Profile: cfg.ProfileName,
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
