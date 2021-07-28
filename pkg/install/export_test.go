package install

import "github.com/weaveworks/pctl/pkg/install/artifact"

func (i *Installer) SetWriter(b artifact.ArtifactWriter) {
	i.artifactWriter = b
}
