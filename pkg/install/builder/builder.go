package builder

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/runtime/dependency"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	"sigs.k8s.io/kustomize/api/types"

	"github.com/weaveworks/pctl/pkg/install/artifact"
)

const (
	defaultValuesKey           = "default-values.yaml"
	helmChartLocation          = "helm-chart"
	kustomizeWrapperObjectName = "kustomize-flux"
)

// Builder can build an artifacts from an installation and a profile artifact.
//go:generate counterfeiter -o fakes/builder_maker.go . Builder
type Builder interface {
	// Build a single artifact from a profile artifact and installation.
	Build(artifact profilesv1.Artifact, installation profilesv1.ProfileInstallation, definition profilesv1.ProfileDefinition) ([]artifact.Artifact, error)
}

// Config defines some common configuration values for builders.
type Config struct {
	GitRepositoryName      string
	GitRepositoryNamespace string
	RootDir                string
}

// ArtifactBuilder will build helm chart resources.
type ArtifactBuilder struct {
	Config
	artifact *artifact.Artifact
}

var _ Builder = &ArtifactBuilder{}

// Build a single artifact from a profile artifact and installation.
func (c *ArtifactBuilder) Build(partifact profilesv1.Artifact, installation profilesv1.ProfileInstallation, definition profilesv1.ProfileDefinition) ([]artifact.Artifact, error) {
	var deps []profilesv1.Artifact
	for _, dep := range partifact.DependsOn {
		d, ok := c.containsArtifact(dep.Name, definition.Spec.Artifacts)
		if !ok {
			return nil, fmt.Errorf("%s's depending artifact %s not found in the list of artifacts", partifact.Name, dep.Name)
		}
		deps = append(deps, d)
	}
	c.artifact = &artifact.Artifact{Name: partifact.Name, Kustomize: artifact.Kustomize{
		ObjectWrapper: &types.Kustomization{
			Resources: []string{"kustomize-flux.yaml"},
		},
	}}
	if partifact.Chart != nil {
		if err := c.updateArtifactWithChartDetails(partifact, installation, definition, deps); err != nil {
			return nil, err
		}
	} else if partifact.Kustomize != nil {
		if err := c.updateArtifactWithKustomizationDetails(partifact, installation, definition, deps); err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("no artifact set")
	}
	return []artifact.Artifact{*c.artifact}, nil
}

func (c *ArtifactBuilder) updateArtifactWithChartDetails(partifact profilesv1.Artifact, installation profilesv1.ProfileInstallation, definition profilesv1.ProfileDefinition, deps []profilesv1.Artifact) error {
	if err := c.validateChartArtifact(partifact); err != nil {
		return fmt.Errorf("validation failed for artifact %s: %w", partifact.Name, err)
	}
	c.artifact.SubFolder = helmChartLocation
	helmRelease, cfgMap := c.makeHelmReleaseObjects(partifact, installation, definition.Name)
	if cfgMap != nil {
		c.artifact.Objects = append(c.artifact.Objects, artifact.Object{Object: cfgMap, Path: helmChartLocation})
	}
	c.artifact.Objects = append(c.artifact.Objects, artifact.Object{Object: helmRelease, Path: helmChartLocation})
	if partifact.Chart.Path != "" {
		if c.GitRepositoryNamespace == "" && c.GitRepositoryName == "" {
			return fmt.Errorf("in case of local resources, the flux gitrepository object's details must be provided")
		}
		helmRelease.Spec.Chart.Spec.Chart = filepath.Join(c.RootDir, "artifacts", partifact.Name, "helm-chart", partifact.Chart.Path)
		branch := installation.Spec.Source.Branch
		if installation.Spec.Source.Tag != "" {
			branch = installation.Spec.Source.Tag
		}
		c.artifact.RepoURL = installation.Spec.Source.URL
		c.artifact.SparseFolder = definition.Name
		c.artifact.Branch = branch
		c.artifact.PathsToCopy = append(c.artifact.PathsToCopy, partifact.Chart.Path)
		c.artifact.Kustomize.LocalResourceLimiter = &types.Kustomization{
			Resources: []string{"HelmRelease.yaml"},
		}
	}
	if partifact.Chart.URL != "" {
		helmRepository := c.makeHelmRepository(partifact.Chart.URL, partifact.Chart.Name, installation)
		c.artifact.Objects = append(c.artifact.Objects, artifact.Object{Object: helmRepository, Path: helmChartLocation})
	}
	kustomizationWrapper := c.makeKustomizeHelmReleaseWrapper(partifact, installation, definition.Name, deps)
	c.artifact.Objects = append(c.artifact.Objects, artifact.Object{Object: kustomizationWrapper, Name: kustomizeWrapperObjectName})
	return nil
}

func (c *ArtifactBuilder) updateArtifactWithKustomizationDetails(partifact profilesv1.Artifact, installation profilesv1.ProfileInstallation, definition profilesv1.ProfileDefinition, deps []profilesv1.Artifact) error {
	if c.GitRepositoryNamespace == "" && c.GitRepositoryName == "" {
		return fmt.Errorf("in case of local resources, the flux gitrepository object's details must be provided")
	}
	if err := c.validateKustomizeArtifact(partifact); err != nil {
		return fmt.Errorf("validation failed for artifact %s: %w", partifact.Name, err)
	}
	branch := installation.Spec.Source.Branch
	if installation.Spec.Source.Tag != "" {
		branch = installation.Spec.Source.Tag
	}
	c.artifact.RepoURL = installation.Spec.Source.URL
	c.artifact.SparseFolder = definition.Name
	c.artifact.Branch = branch
	c.artifact.PathsToCopy = append(c.artifact.PathsToCopy, partifact.Kustomize.Path)
	path := filepath.Join(c.RootDir, "artifacts", partifact.Name, partifact.Kustomize.Path)
	wrapper := c.makeKustomization(partifact, path, installation, definition.Name, deps)
	c.artifact.Objects = append(c.artifact.Objects, artifact.Object{Object: wrapper, Name: kustomizeWrapperObjectName})
	return nil
}

// validateChartArtifact validates that the artifact has valid chart properties.
func (c *ArtifactBuilder) validateChartArtifact(in profilesv1.Artifact) error {
	if in.Profile != nil {
		return apis.ErrMultipleOneOf("chart", "profile")
	}
	if in.Kustomize != nil {
		return apis.ErrMultipleOneOf("chart", "kustomize")
	}
	if in.Chart.Path != "" && in.Chart.URL != "" {
		return apis.ErrMultipleOneOf("chart.path", "chart.url")
	}
	return nil
}

func (c *ArtifactBuilder) makeHelmReleaseObjects(artifact profilesv1.Artifact, installation profilesv1.ProfileInstallation, definitionName string) (*helmv2.HelmRelease, *corev1.ConfigMap) {
	var helmChartSpec helmv2.HelmChartTemplateSpec
	if artifact.Chart.Path != "" {
		helmChartSpec = c.makeGitChartSpec(path.Join(installation.Spec.Source.Path, artifact.Chart.Path))
	} else if artifact.Chart != nil {
		helmChartSpec = c.makeHelmChartSpec(artifact.Chart.Name, artifact.Chart.Version, installation)
	}
	var (
		cfgMap *corev1.ConfigMap
		values []helmv2.ValuesReference
	)
	if artifact.Chart.DefaultValues != "" {
		cfgMap = c.makeDefaultValuesCfgMap(artifact.Name, artifact.Chart.DefaultValues, installation)
		// the default values always need to be at index 0
		values = append(values, helmv2.ValuesReference{
			Kind:      "ConfigMap",
			Name:      cfgMap.Name,
			ValuesKey: defaultValuesKey,
		})
	}
	if installation.Spec.ConfigMap != "" {
		artifactNameParts := strings.Split(artifact.Name, "/")
		values = append(values, helmv2.ValuesReference{
			Kind:      "ConfigMap",
			Name:      installation.Spec.ConfigMap,
			ValuesKey: artifactNameParts[len(artifactNameParts)-1],
		})
	}
	helmRelease := &helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.makeArtifactName(artifact.Name, installation.Name, definitionName),
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
			ValuesFrom: values,
		},
	}
	return helmRelease, cfgMap
}

func (c *ArtifactBuilder) makeHelmRepository(url string, name string, installation profilesv1.ProfileInstallation) *sourcev1.HelmRepository {
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

func (c *ArtifactBuilder) makeHelmRepoName(name string, installation profilesv1.ProfileInstallation) string {
	repoParts := strings.Split(installation.Spec.Source.URL, "/")
	repoName := repoParts[len(repoParts)-1]
	return c.join(installation.Name, repoName, name)
}

func (c *ArtifactBuilder) makeKustomizeHelmReleaseWrapper(artifact profilesv1.Artifact, installation profilesv1.ProfileInstallation, definitionName string, dependencies []profilesv1.Artifact) *kustomizev1.Kustomization {
	path := filepath.Join(c.RootDir, "artifacts", artifact.Name, "helm-chart")
	wrapper := c.makeKustomization(artifact, path, installation, definitionName, dependencies)
	wrapper.Spec.HealthChecks = []meta.NamespacedObjectKindReference{
		{
			APIVersion: helmv2.GroupVersion.String(),
			Kind:       helmv2.HelmReleaseKind,
			Name:       c.makeArtifactName(artifact.Name, installation.Name, definitionName),
			Namespace:  installation.ObjectMeta.Namespace,
		},
	}
	return wrapper
}

func (c *ArtifactBuilder) makeGitChartSpec(path string) helmv2.HelmChartTemplateSpec {
	return helmv2.HelmChartTemplateSpec{
		Chart: path,
		SourceRef: helmv2.CrossNamespaceObjectReference{
			Kind:      sourcev1.GitRepositoryKind,
			Name:      c.GitRepositoryName,
			Namespace: c.GitRepositoryNamespace,
		},
	}
}

func (c *ArtifactBuilder) makeHelmChartSpec(chart string, version string, installation profilesv1.ProfileInstallation) helmv2.HelmChartTemplateSpec {
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

func (c *ArtifactBuilder) makeDefaultValuesCfgMap(name, data string, installation profilesv1.ProfileInstallation) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.makeCfgMapName(name, installation),
			Namespace: installation.ObjectMeta.Namespace,
		},
		Data: map[string]string{
			defaultValuesKey: data,
		},
	}
}

func (c *ArtifactBuilder) makeCfgMapName(name string, installation profilesv1.ProfileInstallation) string {
	if strings.Contains(name, "/") {
		name = filepath.Base(name)
	}
	return c.join(installation.Name, name, "defaultvalues")
}

// containsArtifact checks whether an artifact with a specific name exists in a list of artifacts.
func (c *ArtifactBuilder) containsArtifact(name string, stack []profilesv1.Artifact) (profilesv1.Artifact, bool) {
	for _, a := range stack {
		if a.Name == name {
			return a, true
		}
	}
	return profilesv1.Artifact{}, false
}

// makeArtifactName creates a name for an artifact.
func (c *ArtifactBuilder) makeArtifactName(name string, installationName, definitionName string) string {
	// if this is a nested artifact, it's name contains a /
	if strings.Contains(name, "/") {
		name = filepath.Base(name)
	}
	return c.join(installationName, definitionName, name)
}

// join creates a joined string of name using - as a join character.
func (c *ArtifactBuilder) join(s ...string) string {
	return strings.Join(s, "-")
}

// makeKustomization creates a Kustomize object.
func (c *ArtifactBuilder) makeKustomization(
	artifact profilesv1.Artifact,
	repoPath string,
	installation profilesv1.ProfileInstallation,
	definitionName string,
	dependencies []profilesv1.Artifact) *kustomizev1.Kustomization {
	var dependsOn []dependency.CrossNamespaceDependencyReference
	for _, dep := range dependencies {
		dependsOn = append(dependsOn, dependency.CrossNamespaceDependencyReference{
			Name:      c.makeArtifactName(dep.Name, installation.Name, definitionName),
			Namespace: installation.Namespace,
		})
	}
	return &kustomizev1.Kustomization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.makeArtifactName(artifact.Name, installation.Name, definitionName),
			Namespace: installation.ObjectMeta.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       kustomizev1.KustomizationKind,
			APIVersion: kustomizev1.GroupVersion.String(),
		},
		Spec: kustomizev1.KustomizationSpec{
			Path:            repoPath,
			Interval:        metav1.Duration{Duration: time.Minute * 5},
			Prune:           true,
			TargetNamespace: installation.ObjectMeta.Namespace,
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind:      sourcev1.GitRepositoryKind,
				Name:      c.GitRepositoryName,
				Namespace: c.GitRepositoryNamespace,
			},
			DependsOn: dependsOn,
		},
	}
}

// validateKustomizeArtifact validates that the artifact has valid kustomize properties.
func (c *ArtifactBuilder) validateKustomizeArtifact(in profilesv1.Artifact) error {
	if in.Chart != nil {
		return apis.ErrMultipleOneOf("chart", "kustomize")
	}
	if in.Profile != nil {
		return apis.ErrMultipleOneOf("profile", "kustomize")
	}
	return nil
}
