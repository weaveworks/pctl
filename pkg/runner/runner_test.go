package runner_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/weaveworks/kivo-cli/pkg/runner"
)

var _ = Describe("runner", func() {
	When("there are commands to be executed", func() {
		It("can run existing commands", func() {
			r := runner.CLIRunner{}
			out, err := r.Run("echo", "this is the output")
			Expect(err).ToNot(HaveOccurred())
			Expect(string(out)).To(Equal("this is the output\n"))
		})
		It("can run them even if there are no arguments", func() {
			r := runner.CLIRunner{}
			out, err := r.Run("echo")
			Expect(err).ToNot(HaveOccurred())
			Expect(string(out)).To(Equal("\n"))
		})
	})
	When("the command does not exist", func() {
		It("returns a sensible error", func() {
			r := runner.CLIRunner{}
			out, err := r.Run("doesnotexist", "args1")
			Expect(errors.Unwrap(err)).To(MatchError("executable file not found in $PATH"))
			Expect(string(out)).To(Equal(""))
		})
	})
})
