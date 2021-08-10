package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/weaveworks/pctl/pkg/bootstrap"
)

func bootstrapCmd() *cli.Command {
	return &cli.Command{
		Name:      "bootstrap",
		Usage:     "bootstrap local git repository",
		UsageText: "pctl bootstrap",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "git-repository",
				Value: "",
				Usage: "The namespace and name of the GitRepository object governing the flux repo.",
			},
		},
		Action: func(c *cli.Context) error {
			directory, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to fetch current working directory: %w", err)
			}
			if c.Args().Len() > 0 {
				directory = c.Args().First()
			}

			gitRepository := c.String("git-repository")
			split := strings.Split(gitRepository, "/")
			if len(split) != 2 {
				return fmt.Errorf("git-repository must in format <namespace>/<name>; was: %s", gitRepository)
			}
			gitRepoNamespace := split[0]
			gitRepoName := split[1]

			if err := bootstrap.CreateConfig(gitRepoNamespace, gitRepoName, directory); err != nil {
				return fmt.Errorf("failed to bootstrap: %w", err)
			}
			logger.Successf("bootstrap completed")
			return nil
		},
	}
}
