package install

import "github.com/weaveworks/pctl/pkg/install/builder"

func (i *Installer) SetBuilder(b builder.Builder) {
	i.Builder = b
}
