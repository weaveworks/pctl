package bootstrap

import "github.com/weaveworks/kivo-cli/pkg/runner"

func SetRunner(r2 runner.Runner) {
	r = r2
}
