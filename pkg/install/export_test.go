package install

import "github.com/weaveworks/kivo-cli/pkg/install/artifact"

func (i *Installer) SetWriter(b artifact.ArtifactWriter) {
	i.artifactWriter = b
}
