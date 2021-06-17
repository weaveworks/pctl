package profile

import (
	"path"
	"path/filepath"
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const defaultValuesKey = "default-values.yaml"

func (p *Profile) makeHelmRepository(url string, name string) *sourcev1.HelmRepository {
	return &sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.makeHelmRepoName(name),
			Namespace: p.installation.ObjectMeta.Namespace,
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

func (p *Profile) makeHelmRepoName(name string) string {
	repoParts := strings.Split(p.installation.Spec.Source.URL, "/")
	repoName := repoParts[len(repoParts)-1]
	return join(p.installation.Name, repoName, name)
}

func (p *Profile) makeHelmReleaseObjects(artifact profilesv1.Artifact, repoPath string) (*helmv2.HelmRelease, *corev1.ConfigMap) {
	var helmChartSpec helmv2.HelmChartTemplateSpec
	if artifact.Chart.Path != "" {
		helmChartSpec = p.makeGitChartSpec(path.Join(repoPath, artifact.Chart.Path))
	} else if artifact.Chart != nil {
		helmChartSpec = p.makeHelmChartSpec(artifact.Chart.Name, artifact.Chart.Version)
	}

	var (
		cfgMap *corev1.ConfigMap
		values []helmv2.ValuesReference
	)
	if artifact.Chart.DefaultValues != "" {
		cfgMap = p.makeDefaultValuesCfgMap(artifact.Name, artifact.Chart.DefaultValues)
		// the default values always need to be at index 0
		values = []helmv2.ValuesReference{{
			Kind:      "ConfigMap",
			Name:      cfgMap.Name,
			ValuesKey: defaultValuesKey,
		}}
	}
	values = append(values, p.installation.Spec.ValuesFrom...)

	helmRelease := &helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.makeArtifactName(artifact.Name),
			Namespace: p.installation.ObjectMeta.Namespace,
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

func (p *Profile) makeGitChartSpec(path string) helmv2.HelmChartTemplateSpec {
	return helmv2.HelmChartTemplateSpec{
		Chart: path,
		SourceRef: helmv2.CrossNamespaceObjectReference{
			Kind:      sourcev1.GitRepositoryKind,
			Name:      p.gitRepositoryName,
			Namespace: p.gitRepositoryNamespace,
		},
	}
}

func (p *Profile) makeHelmChartSpec(chart string, version string) helmv2.HelmChartTemplateSpec {
	return helmv2.HelmChartTemplateSpec{
		Chart: chart,
		SourceRef: helmv2.CrossNamespaceObjectReference{
			Kind:      sourcev1.HelmRepositoryKind,
			Name:      p.makeHelmRepoName(chart),
			Namespace: p.installation.ObjectMeta.Namespace,
		},
		Version: version,
	}
}

func (p *Profile) makeDefaultValuesCfgMap(name, data string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.makeCfgMapName(name),
			Namespace: p.installation.ObjectMeta.Namespace,
		},
		Data: map[string]string{
			defaultValuesKey: data,
		},
	}
}

func (p *Profile) makeCfgMapName(name string) string {
	if strings.Contains(name, "/") {
		name = filepath.Base(name)
	}
	return join(p.installation.Name, name, "defaultvalues")
}
