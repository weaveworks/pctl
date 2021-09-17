package main

import (
	"flag"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli/v2"

	"github.com/weaveworks/pctl/pkg/bootstrap"
)

var _ = Describe("add", func() {
	Context("getOutFolder", func() {
		It("returns what the user has set", func() {
			add := addCmd()
			f := &flag.FlagSet{}
			f.String("out", "user-defined", "")
			_ = f.Set("out", "user-defined")
			c := cli.NewContext(&cli.App{
				Commands: []*cli.Command{add},
			}, f, nil)
			out, err := getProfileOutputDirectory(c, nil)
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
				out, err := getProfileOutputDirectory(c, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(out).To(Equal(defaultOut))
			})
		})
		When("the user has something saved in the config file", func() {
			It("will return the saved values", func() {
				add := addCmd()
				f := &flag.FlagSet{}
				f.String("out", "user-defined", "")
				c := cli.NewContext(&cli.App{
					Commands: []*cli.Command{add},
				}, f, nil)
				out, err := getProfileOutputDirectory(c, &bootstrap.Config{
					DefaultDir: "config-dir",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(out).To(Equal("config-dir"))
			})
		})
		When("the user has a config file and overrides it with explicit setting", func() {
			It("will return the saved values", func() {
				add := addCmd()
				f := &flag.FlagSet{}
				f.String("out", "user-defined", "")
				_ = f.Set("out", "overwrite")
				c := cli.NewContext(&cli.App{
					Commands: []*cli.Command{add},
				}, f, nil)
				out, err := getProfileOutputDirectory(c, &bootstrap.Config{
					DefaultDir: "user-defined",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(out).To(Equal("overwrite"))
			})
		})
	})
})
