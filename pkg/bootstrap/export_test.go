package bootstrap

import "github.com/weaveworks/pctl/pkg/runner"

func SetRunner(r2 runner.Runner) {
	r = r2
}
