package main

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestProfile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Generate Suite")
}
