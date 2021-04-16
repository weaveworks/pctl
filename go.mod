module github.com/weaveworks/pctl

go 1.16

require (
	github.com/fluxcd/helm-controller/api v0.9.0 // indirect
	github.com/fluxcd/kustomize-controller/api v0.11.0 // indirect
	github.com/fluxcd/source-controller/api v0.11.0 // indirect
	github.com/go-logr/logr v0.4.0
	github.com/onsi/ginkgo v1.16.1
	github.com/onsi/gomega v1.11.0
	github.com/urfave/cli/v2 v2.3.0
	github.com/weaveworks/profiles v0.0.0-20210409161211-27f03f793ca7
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.20.5
)

replace github.com/weaveworks/profiles => /Users/skarlso/weaveworks/cloud/src/profiles
