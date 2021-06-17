package chart

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/weaveworks/pctl/pkg/profile/artifact"
)

// Config defines some common configuration values for builders.
type Config struct {
	GitRepositoryName      string
	GitRepositoryNamespace string
	RootDir                string
}

// Builder will build helm chart resources.
type Builder struct {
	Config
}

// Build a single artifact from a profile artifact and installation.
func (c *Builder) Build(att profilesv1.Artifact, installation profilesv1.ProfileInstallation, definition profilesv1.ProfileDefinition) ([]artifact.Artifact, error) {
	a := artifact.Artifact{Name: att.Name}
	helmRelease := c.makeHelmRelease(att, installation, definition.Name)
	a.Objects = append(a.Objects, helmRelease)
	if att.Chart.Path != "" {
		if c.GitRepositoryNamespace == "" && c.GitRepositoryName == "" {
			return nil, fmt.Errorf("in case of local resources, the flux gitrepository object's details must be provided")
		}
		helmRelease.Spec.Chart.Spec.Chart = filepath.Join(c.RootDir, "artifacts", att.Name, att.Chart.Path)
		branch := installation.Spec.Source.Branch
		if installation.Spec.Source.Tag != "" {
			branch = installation.Spec.Source.Tag
		}
		a.RepoURL = installation.Spec.Source.URL
		a.SparseFolder = definition.Name
		a.Branch = branch
		a.PathsToCopy = append(a.PathsToCopy, att.Chart.Path)
	}
	if att.Chart.URL != "" {
		helmRepository := c.makeHelmRepository(att.Chart.URL, att.Chart.Name, installation)
		a.Objects = append(a.Objects, helmRepository)
	}
	return []artifact.Artifact{a}, nil
}

func (c *Builder) makeHelmRelease(artifact profilesv1.Artifact, installation profilesv1.ProfileInstallation, definitionName string) *helmv2.HelmRelease {
	var helmChartSpec helmv2.HelmChartTemplateSpec
	if artifact.Chart.Path != "" {
		helmChartSpec = c.makeGitChartSpec(path.Join(installation.Spec.Source.Path, artifact.Chart.Path))
	} else if artifact.Chart != nil {
		helmChartSpec = c.makeHelmChartSpec(artifact.Chart.Name, artifact.Chart.Version, installation)
	}
	helmRelease := &helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      makeArtifactName(artifact.Name, installation.Name, definitionName),
			Namespace: installation.ObjectMeta.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       helmv2.HelmReleaseKind,
			APIVersion: helmv2.GroupVersion.String(),
		},
		Spec: helmv2.HelmReleaseSpec{
			Chart: helmv2.HelmChartTemplate{
				Spec: helmChartSpec,
			},
			Values:     installation.Spec.Values,
			ValuesFrom: installation.Spec.ValuesFrom,
		},
	}
	return helmRelease
}

func (c *Builder) makeHelmRepository(url string, name string, installation profilesv1.ProfileInstallation) *sourcev1.HelmRepository {
	return &sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.makeHelmRepoName(name, installation),
			Namespace: installation.ObjectMeta.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       sourcev1.HelmRepositoryKind,
			APIVersion: sourcev1.GroupVersion.String(),
		},
		Spec: sourcev1.HelmRepositorySpec{
			URL: url,
		},
	}
}

func (c *Builder) makeHelmRepoName(name string, installation profilesv1.ProfileInstallation) string {
	repoParts := strings.Split(installation.Spec.Source.URL, "/")
	repoName := repoParts[len(repoParts)-1]
	return join(installation.Name, repoName, name)
}

func makeArtifactName(name string, installationName, definitionName string) string {
	// if this is a nested artifact, it's name contains a /
	if strings.Contains(name, "/") {
		name = filepath.Base(name)
	}
	return join(installationName, definitionName, name)
}

func join(s ...string) string {
	return strings.Join(s, "-")
}

func (c *Builder) makeGitChartSpec(path string) helmv2.HelmChartTemplateSpec {
	return helmv2.HelmChartTemplateSpec{
		Chart: path,
		SourceRef: helmv2.CrossNamespaceObjectReference{
			Kind:      sourcev1.GitRepositoryKind,
			Name:      c.GitRepositoryName,
			Namespace: c.GitRepositoryNamespace,
		},
	}
}

func (c *Builder) makeHelmChartSpec(chart string, version string, installation profilesv1.ProfileInstallation) helmv2.HelmChartTemplateSpec {
	return helmv2.HelmChartTemplateSpec{
		Chart: chart,
		SourceRef: helmv2.CrossNamespaceObjectReference{
			Kind:      sourcev1.HelmRepositoryKind,
			Name:      c.makeHelmRepoName(chart, installation),
			Namespace: installation.ObjectMeta.Namespace,
		},
		Version: version,
	}
}
