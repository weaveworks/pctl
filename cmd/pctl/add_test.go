package main

import (
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
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
			_ = f.Set("out", "user-defined")
			c := cli.NewContext(&cli.App{
				Commands: []*cli.Command{add},
			}, f, nil)
			out, err := getOutFolder(c)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal("user-defined"))
		})
		When("the user didn't provide a value or use a config file", func() {
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
		When("the user has something saved in the config file", func() {
			BeforeEach(func() {
				err := os.MkdirAll(filepath.Join(tmp, ".pctl"), 0700)
				Expect(err).ToNot(HaveOccurred())
				err = ioutil.WriteFile(filepath.Join(tmp, ".pctl", "config.yaml"), []byte("defaultDir: config-dir"), 0655)
				Expect(err).ToNot(HaveOccurred())
				_ = os.Chdir(tmp)
				cmd := exec.Command("git", "init", tmp)
				err = cmd.Run()
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
