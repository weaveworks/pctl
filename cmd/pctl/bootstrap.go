package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/bootstrap"
	"github.com/weaveworks/pctl/pkg/log"
)

func bootstrapCmd() *cli.Command {
	return &cli.Command{
		Name:      "bootstrap",
		Usage:     "bootstrap default settings for pctl",
		UsageText: "pctl bootstrap",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "git-repository",
				Value: "",
				Usage: "The namespace and name of the GitRepository object governing the flux repo.",
			},
			&cli.StringFlag{
				Name:  "default-dir",
				Value: "",
				Usage: "Directory to always use with pctl add.",
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

			defaultDir := c.String("default-dir")
			if err := bootstrap.CreateConfig(bootstrap.Config{
				GitRepository: profilesv1.GitRepository{
					Name:      gitRepoName,
					Namespace: gitRepoNamespace,
				},
				DefaultDir: defaultDir,
			}, directory); err != nil {
				return fmt.Errorf("failed to bootstrap: %w", err)
			}
			log.Successf("bootstrap completed")
			return nil
		},
	}
}
