package spike_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSpike(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Spike Suite")
}
