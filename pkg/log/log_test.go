package log_test

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/log"
)

var _ = Describe("PrintLogger", func() {
	var (
		logger = log.PrintLogger{}
		r      *os.File
		w      *os.File
		tmp    *os.File
	)

	BeforeEach(func() {
		r, w, _ = os.Pipe()
		tmp = os.Stdout
		os.Stdout = w
	})

	It("logs a message to stdout with a tickmark", func() {
		defer func() {
			os.Stdout = tmp
		}()

		logger.Warningf("test-warning")
		_ = w.Close()

		stdout, _ := ioutil.ReadAll(r)
		Expect(string(stdout)).To(Equal("⚠️ test-warning\n"))
	})
})
