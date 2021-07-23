package install

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/install/artifact"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// ProfileInstaller installs the profile
//go:generate counterfeiter -o fakes/fake_profile_installer.go . ProfileInstaller
type ProfileInstaller interface {
	Install(installation profilesv1.ProfileInstallation) error
}

//Ensure Installer implements ProfileInstaller interface
var _ ProfileInstaller = &Installer{}

// Config defines configurable options for the installer
type Config struct {
	GitClient        git.Git
	RootDir          string
	GitRepoNamespace string
	GitRepoName      string
}

type Installer struct {
	Config
	clonedRepos map[string]string
	Writer      artifact.ArtifactWriter
}

// NewInstaller creates a new profiles installer
func NewInstaller(cfg Config) *Installer {
	return &Installer{
		clonedRepos: make(map[string]string),
		Config:      cfg,
		Writer: &artifact.Writer{
			GitRepositoryName:      cfg.GitRepoName,
			GitRepositoryNamespace: cfg.GitRepoNamespace,
			RootDir:                cfg.RootDir,
		},
	}
}

func (i *Installer) Install(installation profilesv1.ProfileInstallation) error {
	artifacts, err := i.composeArtifacts(installation, false)
	if err != nil {
		return err
	}
	return i.Writer.Write(installation, artifacts)
}

func (i *Installer) composeArtifacts(installation profilesv1.ProfileInstallation, nested bool) ([]artifact.ArtifactWrapper, error) {
	path := installation.Spec.Source.Path
	branchOrTag := installation.Spec.Source.Tag
	if installation.Spec.Source.Tag == "" {
		branchOrTag = installation.Spec.Source.Branch
	}
	profileDef, err := i.GetProfileDefinition(installation.Spec.Source.URL, branchOrTag, path)
	if err != nil {
		return nil, err
	}
	profileRepoKey := cloneCacheKey(installation.Spec.Source.URL, branchOrTag)

	var artifacts []artifact.ArtifactWrapper
	for _, a := range profileDef.Spec.Artifacts {
		if a.Profile != nil {
			nestedInstallation := installation.DeepCopyObject().(*profilesv1.ProfileInstallation)
			nestedInstallation.Spec.Source.URL = a.Profile.Source.URL
			nestedInstallation.Spec.Source.Branch = a.Profile.Source.Branch
			nestedInstallation.Spec.Source.Tag = a.Profile.Source.Tag
			nestedInstallation.Spec.Source.Path = a.Profile.Source.Path
			if a.Profile.Source.Tag != "" {
				path := "."
				splitTag := strings.Split(a.Profile.Source.Tag, "/")
				if len(splitTag) > 1 {
					path = splitTag[0]
				}
				nestedInstallation.Spec.Source.Path = path
			}
			nestedInstallation.Name = a.Name
			nestedArtifacts, err := i.composeArtifacts(*nestedInstallation, true)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, nestedArtifacts...)
		} else {
			if nested {
				artifacts = append(artifacts, artifact.ArtifactWrapper{
					Artifact:                  a,
					NestedProfileArtifactName: installation.Name,
					PathToProfileClone:        filepath.Join(i.clonedRepos[profileRepoKey], installation.Spec.Source.Path),
					ProfileName:               profileDef.Name,
				})
			} else {
				artifacts = append(artifacts, artifact.ArtifactWrapper{
					Artifact:           a,
					PathToProfileClone: filepath.Join(i.clonedRepos[profileRepoKey], installation.Spec.Source.Path),
					ProfileName:        profileDef.Name,
				})
			}
		}
	}
	return artifacts, nil
}

func (i *Installer) GetProfileDefinition(repoURL, branch, path string) (profilesv1.ProfileDefinition, error) {
	// Add postfix so potential nested profiles don't clone into the same folder.
	u, err := uuid.NewRandom()
	if err != nil {
		return profilesv1.ProfileDefinition{}, err
	}
	// this should not be possible, but I don't like leaving open spots for an index overflow
	if len(u.String()) < 7 {
		return profilesv1.ProfileDefinition{}, errors.New("the generated uuid is not long enough")
	}

	var (
		tmp string
	)
	if v, ok := i.clonedRepos[cloneCacheKey(repoURL, branch)]; ok {
		tmp = v
	} else {
		px := u.String()[:6]
		tmp, err = ioutil.TempDir("", "cloned_profile"+px)
		if err != nil {
			return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to create temp folder for cloning repository: %w", err)
		}
		if err := i.GitClient.Clone(repoURL, branch, tmp); err != nil {
			return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to clone repo %q: %w", repoURL, err)
		}
		i.clonedRepos[cloneCacheKey(repoURL, branch)] = tmp
	}

	content, err := ioutil.ReadFile(filepath.Join(tmp, path, "profile.yaml"))
	if err != nil {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to read profile.yaml in repo %q branch %q path %q: %w", repoURL, branch, path, err)
	}

	profile := profilesv1.ProfileDefinition{}
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 4096).Decode(&profile)
	if err != nil {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to parse profile.yaml: %w", err)
	}

	return profile, nil
}

func cloneCacheKey(url, branch string) string {
	return fmt.Sprintf("%s:%s", url, branch)
}
