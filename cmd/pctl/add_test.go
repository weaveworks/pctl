package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli/v2"
)

var _ = Describe("add", func() {
	Context("getOutFolder", func() {
		var (
			tmp string
			err error
		)
		BeforeEach(func() {
			tmp, err = ioutil.TempDir("", "get-out-folder")
			Expect(err).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			_ = os.RemoveAll(tmp)
		})
		It("returns what the user has set", func() {
			add := addCmd()
			f := &flag.FlagSet{}
			f.String("out", "user-defined", "")
			f.Set("out", "user-defined")
			c := cli.NewContext(&cli.App{
				Commands: []*cli.Command{add},
			}, f, nil)
			out, err := getOutFolder(c)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal("user-defined"))
		})
		When("the user didn't set anything", func() {
			It("returns the default", func() {
				add := addCmd()
				f := &flag.FlagSet{}
				f.String("out", "user-defined", "")
				c := cli.NewContext(&cli.App{
					Commands: []*cli.Command{add},
				}, f, nil)
				out, err := getOutFolder(c)
				Expect(err).ToNot(HaveOccurred())
				Expect(out).To(Equal(defaultOut))
			})
		})
		// NOTE: Doesn't work right now because the working directory is not what it should be.
		XWhen("the user has something saved in the config file", func() {
			BeforeEach(func() {
				err := os.MkdirAll(".pctl", 0700)
				Expect(err).ToNot(HaveOccurred())
				err = ioutil.WriteFile(filepath.Join(".pctl", "config"), []byte("defaultDir: config-dir"), 0655)
				Expect(err).ToNot(HaveOccurred())
			})
			It("will return the saved values", func() {
				add := addCmd()
				f := &flag.FlagSet{}
				f.String("out", "user-defined", "")
				c := cli.NewContext(&cli.App{
					Commands: []*cli.Command{add},
				}, f, nil)
				out, err := getOutFolder(c)
				Expect(err).ToNot(HaveOccurred())
				Expect(out).To(Equal("config-dir"))
			})
		})
	})
})
