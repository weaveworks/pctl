package install

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/weaveworks/pctl/pkg/install/artifact"
	"github.com/weaveworks/pctl/pkg/install/builder"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type Installer2 struct {
	Config
	clonedRepos map[string]string
	Builder     builder.Builder
}

// NewInstaller creates a new profiles installer
func NewInstaller2(cfg Config) *Installer2 {
	return &Installer2{
		clonedRepos: make(map[string]string),
		Config:      cfg,
	}
}

func (i *Installer2) Install(installation profilesv1.ProfileInstallation) error {
	artifacts, err := i.flatternArtifacts(installation, false)
	for _, a := range artifacts {
		fmt.Println("---------------------------------")
		fmt.Println("name: ", a.Name)
		fmt.Println("profileName: ", a.ProfileName)
		fmt.Println("nestedName: ", a.NestedProfileDir)
		fmt.Println("repoKey: ", a.RepoKey)
	}
	if err != nil {
		return err
	}
	builder := &builder.ArtifactBuilder2{Config: builder.Config{
		GitRepositoryName:      i.Config.GitRepoName,
		GitRepositoryNamespace: i.Config.GitRepoNamespace,
		RootDir:                i.Config.RootDir,
	}}
	return builder.BuildAndWrite(installation, artifacts, i.clonedRepos)
}

func (i *Installer2) flatternArtifacts(installation profilesv1.ProfileInstallation, nested bool) ([]artifact.Artifact2, error) {
	fmt.Printf("flattenrArtifact called with %q: \n%v\n---\n", installation.Spec.Catalog.Profile, installation)
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

	var artifacts []artifact.Artifact2
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
			nestedInstallation.Name = filepath.Join(a.Name)
			nestedArtifacts, err := i.flatternArtifacts(*nestedInstallation, true)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, nestedArtifacts...)
		} else {
			if nested {
				artifacts = append(artifacts, artifact.Artifact2{
					Artifact:         a,
					RepoKey:          profileRepoKey,
					NestedProfileDir: installation.Name,
					ProfileName:      installation.Spec.Source.Path,
				})
			} else {
				artifacts = append(artifacts, artifact.Artifact2{
					Artifact:    a,
					RepoKey:     profileRepoKey,
					ProfileName: installation.Spec.Catalog.Profile,
				})
			}
		}
	}
	return artifacts, nil
}

func (i *Installer2) GetProfileDefinition(repoURL, branch, path string) (profilesv1.ProfileDefinition, error) {
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
			return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to clone the repo: %w", err)
		}
		i.clonedRepos[cloneCacheKey(repoURL, branch)] = tmp
	}

	content, err := ioutil.ReadFile(filepath.Join(tmp, path, "profile.yaml"))
	if err != nil {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("could not find file at cloned location: %w", err)
	}

	profile := profilesv1.ProfileDefinition{}
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 4096).Decode(&profile)
	if err != nil {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to parse profile: %w", err)
	}

	return profile, nil
}
