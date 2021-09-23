package log_test

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/kivo-cli/pkg/log"
)

var _ = Describe("PrintLogger", func() {
	var (
		r   *os.File
		w   *os.File
		tmp *os.File
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

		log.Actionf("test action")
		log.Waitingf("test waiting")
		log.Successf("test success")
		log.Warningf("test warning")
		log.Failuref("test failure")

		_ = w.Close()

		stdout, _ := ioutil.ReadAll(r)
		Expect(string(stdout)).To(Equal(
			`► test action
◎ test waiting
✔ test success
⚠️ WARNING: test warning
✗ test failure
`))
	})
})
