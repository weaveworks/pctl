package catalog

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"

	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/profile"
)

// InstallConfig defines parameters for the installation call.
type InstallConfig struct {
	CatalogClient CatalogClient
	GitClient     git.Git
	ProfileBranch string
	CatalogName   string
	ConfigMap     string
	Namespace     string
	ProfileName   string
	SubName       string
	Version       string
	Directory     string
	URL           string
	Path          string
	GitRepository string
}

// MakeArtifacts returns artifacts for a subscription
type MakeArtifacts func(sub profilesv1.ProfileSubscription, gitClient git.Git, repoRoot, gitRepoNamespace, gitRepoName string) ([]profile.Artifact, error)

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
	var (
		gitRepoNamespace string
		gitRepoName      string
	)
	if cfg.GitRepository != "" {
		split := strings.Split(cfg.GitRepository, "/")
		if len(split) != 2 {
			return fmt.Errorf("git-repository must in format <namespace>/<name>; was: %s", cfg.GitRepository)
		}
		gitRepoNamespace = split[0]
		gitRepoName = split[1]
	}
	artifacts, err := makeArtifacts(subscription, cfg.GitClient, cfg.Directory, gitRepoNamespace, gitRepoName)
	if err != nil {
		return fmt.Errorf("failed to generate artifacts: %w", err)
	}

	profileRootdir := filepath.Join(cfg.Directory, cfg.ProfileName)
	artifactsRootDir := filepath.Join(profileRootdir, "artifacts")

	for _, artifact := range artifacts {
		artifactDir := filepath.Join(artifactsRootDir, artifact.Name)
		if err = os.MkdirAll(artifactDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory")
		}
		for _, obj := range artifact.Objects {
			if obj.GetObjectKind().GroupVersionKind().Kind == sourcev1.GitRepositoryKind {
				// clone and copy
				if g, ok := obj.(*sourcev1.GitRepository); ok {
					tmp, err := ioutil.TempDir("", "sparse_clone_git_repo_"+artifact.Name)
					if err != nil {
						return err
					}
					// artifactsRootDir + artifact.Name + g.Spec.Reference.Tag // helmRelease and the Kustomize -- both who consume a GitRepository object.
					if err := cfg.GitClient.SparseClone(subscription.Spec.ProfileURL, subscription.Spec.Branch, tmp, subscription.Spec.Path); err != nil {
						return fmt.Errorf("failed to sparse clone folder with url: %s; branch: %s; path: %s; with error: %w",
							subscription.Spec.ProfileURL,
							subscription.Spec.Branch,
							subscription.Spec.Path,
							err)
					}

					// nginx
					dir := g.Spec.Reference.Tag
					// nginx/chart/...
					if strings.Contains(dir, string(os.PathSeparator)) {
						dir = filepath.Dir(dir)
					}
					if err := os.Rename(filepath.Join(tmp, subscription.Spec.Path, dir), filepath.Join(artifactDir, dir)); err != nil {
						return fmt.Errorf("failed to move folder: %w", err)
					}
					continue
				}
			}
			filename := filepath.Join(artifactDir, fmt.Sprintf("%s.%s", obj.GetObjectKind().GroupVersionKind().Kind, "yaml"))
			if err := generateOutput(filename, obj); err != nil {
				return err
			}
		}
	}

	return generateOutput(filepath.Join(profileRootdir, "profile.yaml"), &subscription)
}

// getProfileSpec generates a spec based on configured properties.
func getProfileSpec(cfg InstallConfig) (profilesv1.ProfileSubscriptionSpec, error) {
	if cfg.URL != "" {
		return profilesv1.ProfileSubscriptionSpec{
			ProfileURL: cfg.URL,
			Branch:     cfg.ProfileBranch,
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
			Version: p.Version,
			Profile: p.Name,
		},
	}, nil
}

func generateOutput(filename string, o runtime.Object) error {
	e := kjson.NewSerializerWithOptions(kjson.DefaultMetaFactory, nil, nil, kjson.SerializerOptions{Yaml: true, Strict: true})
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
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
