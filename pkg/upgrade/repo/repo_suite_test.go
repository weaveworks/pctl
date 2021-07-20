package repo_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBranch(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Branch Suite")
}
