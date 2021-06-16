package profile

//func (p *Profile) makeHelmRepository(url string, name string) *sourcev1.HelmRepository {
//	return &sourcev1.HelmRepository{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      p.makeHelmRepoName(name),
//			Namespace: p.installation.ObjectMeta.Namespace,
//		},
//		TypeMeta: metav1.TypeMeta{
//			Kind:       sourcev1.HelmRepositoryKind,
//			APIVersion: sourcev1.GroupVersion.String(),
//		},
//		Spec: sourcev1.HelmRepositorySpec{
//			URL: url,
//		},
//	}
//}
//
//func (p *Profile) makeHelmRepoName(name string) string {
//	repoParts := strings.Split(p.installation.Spec.Source.URL, "/")
//	repoName := repoParts[len(repoParts)-1]
//	return join(p.installation.Name, repoName, name)
//}
//
//func (p *Profile) makeGitChartSpec(path string) helmv2.HelmChartTemplateSpec {
//	return helmv2.HelmChartTemplateSpec{
//		Chart: path,
//		SourceRef: helmv2.CrossNamespaceObjectReference{
//			Kind:      sourcev1.GitRepositoryKind,
//			Name:      p.gitRepositoryName,
//			Namespace: p.gitRepositoryNamespace,
//		},
//	}
//}
//
//func (p *Profile) makeHelmChartSpec(chart string, version string) helmv2.HelmChartTemplateSpec {
//	return helmv2.HelmChartTemplateSpec{
//		Chart: chart,
//		SourceRef: helmv2.CrossNamespaceObjectReference{
//			Kind:      sourcev1.HelmRepositoryKind,
//			Name:      p.makeHelmRepoName(chart),
//			Namespace: p.installation.ObjectMeta.Namespace,
//		},
//		Version: version,
//	}
//}
