package main

import (
	"github.com/urfave/cli/v2"

	"github.com/weaveworks/pctl/pkg/cluster"
)

func prepareCmd() *cli.Command {
	return &cli.Command{
		Name:      "prepare",
		Usage:     "prepare an environment with everything profiles need to work",
		UsageText: "pctl prepare",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "If defined, nothing will be applied.",
				Value: false,
			},
			&cli.BoolFlag{
				Name:  "keep",
				Usage: "Keep the downloaded manifest files.",
				Value: false,
			},
			&cli.StringFlag{
				Name:  "version",
				Usage: "Define the tagged version to use which can be found under releases in the profiles repository. Exp: [v]0.0.1",
			},
			&cli.StringFlag{
				Name:        "baseurl",
				Usage:       "Define the url to go and fetch releases from.",
				Value:       releaseUrl,
				DefaultText: releaseUrl,
			},
			&cli.StringFlag{
				Name:  "context",
				Usage: "The Kubernetes context to use to apply the manifest files .",
			},
		},
		Action: func(c *cli.Context) error {
			p, err := cluster.NewPreparer(cluster.PrepConfig{
				BaseURL:     c.String("baseurl"),
				Version:     c.String("version"),
				KubeConfig:  c.String("kubeconfig"),
				KubeContext: c.String("context"),
				DryRun:      c.Bool("dry-run"),
				Keep:        c.Bool("keep"),
			})
			if err != nil {
				return err
			}
			return p.Prepare()
		},
	}
}
