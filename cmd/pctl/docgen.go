package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

func docgenCmd() *cli.Command {
	return &cli.Command{
		Name:      "docgen",
		Usage:     "generate the cli doc pages",
		UsageText: "pctl docgen --out <relative-path-of-dir-to-write-docs>",
		Hidden:    true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "path",
				Usage: "The relative path to write the docs out. Default: pwd.",
			},
		},
		Action: func(c *cli.Context) error {
			outPath := c.String("path")
			if outPath != "" {
				if _, err := os.Stat(outPath); os.IsNotExist(err) {
					if err := os.MkdirAll(outPath, 0755); err != nil {
						return err
					}
				}
			}

			for _, command := range c.App.Commands {
				name := command.Name
				if name == "help" || name == "docgen" {
					continue
				}

				fileName := fmt.Sprintf("pctl-%s-cmd.md", name)
				if err := writeCommandFile(c, name, outPath, fileName); err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func writeCommandFile(c *cli.Context, command, outPath, fileName string) error {
	f, err := os.Create(filepath.Join(outPath, fileName))
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if err := writeHeader(f, command); err != nil {
		return err
	}

	c.App.Writer = f
	_ = cli.ShowCommandHelp(c, command)

	return writeFooter(f)
}

func writeHeader(f *os.File, command string) error {
	if _, err := f.WriteString(fmt.Sprintf("# %s\n\n```\n", command)); err != nil {
		return err
	}

	return nil
}

func writeFooter(f *os.File) error {
	if _, err := f.WriteString("```"); err != nil {
		return err
	}

	return nil
}
