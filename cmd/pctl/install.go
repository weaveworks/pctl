package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/runner"
)

func installCmd() *cli.Command {
	return &cli.Command{
		Name:      "install",
		Usage:     "generate a profile subscription for a profile in a catalog",
		UsageText: "pctl --catalog-url <URL> install --subscription-name pctl-profile --namespace default --branch main --config-secret configmap-name <CATALOG>/<PROFILE>[/<VERSION>]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "subscription-name",
				DefaultText: "pctl-profile",
				Value:       "pctl-profile",
				Usage:       "The name of the subscription.",
			},
			&cli.StringFlag{
				Name:        "namespace",
				DefaultText: "default",
				Value:       "default",
				Usage:       "The namespace to use for generating resources.",
			},
			&cli.StringFlag{
				Name:        "branch",
				Value:       "main",
				DefaultText: "main",
				Usage:       "The branch to use on the repository in which the profile is.",
			},
			&cli.StringFlag{
				Name:  "config-secret",
				Value: "",
				Usage: "The name of the ConfigMap which contains values for this profile.",
			},
			&cli.BoolFlag{
				Name:  "create-pr",
				Value: false,
				Usage: "If given, install will create a PR for the modifications it outputs.",
			},
			&cli.StringFlag{
				Name:        "remote",
				Value:       "origin",
				DefaultText: "origin",
				Usage:       "The remote to push the branch to.",
			},
			&cli.StringFlag{
				Name:        "base",
				Value:       "main",
				DefaultText: "main",
				Usage:       "The base branch to open a PR against.",
			},
			&cli.StringFlag{
				Name:        "out",
				DefaultText: "current",
				Value:       ".",
				Usage:       "Optional location to create the profile installation folder in.",
			},
			&cli.StringFlag{
				Name:  "repo",
				Value: "",
				Usage: "The repository to open a pr against. Format is: org/repo-name",
			},
		},
		Action: func(c *cli.Context) error {
			// Run installation main
			if err := install(c); err != nil {
				return err
			}
			// Create a pull request if desired
			if c.Bool("create-pr") {
				if err := createPullRequest(c); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

// install runs the install part of the `install` command.
func install(c *cli.Context) error {
	profilePath, catalogClient, err := parseArgs(c)
	if err != nil {
		_ = cli.ShowCommandHelp(c, "install")
		return err
	}

	branch := c.String("branch")
	subName := c.String("subscription-name")
	namespace := c.String("namespace")
	configValues := c.String("config-secret")
	dir := c.String("out")

	parts := strings.Split(profilePath, "/")
	if len(parts) < 2 {
		_ = cli.ShowCommandHelp(c, "install")
		return errors.New("both catalog name and profile name must be provided")
	}
	catalogName, profileName := parts[0], parts[1]

	fmt.Printf("generating subscription for profile %s/%s:\n\n", catalogName, profileName)
	cfg := catalog.InstallConfig{
		Branch:        branch,
		CatalogName:   catalogName,
		CatalogClient: catalogClient,
		ConfigMap:     configValues,
		Namespace:     namespace,
		ProfileName:   profileName,
		SubName:       subName,
		Directory:     dir,
	}
	if len(parts) == 3 {
		cfg.Version = parts[2]
	}
	return catalog.Install(cfg)
}

// createPullRequest runs the pull request creation part of the `install` command.
func createPullRequest(c *cli.Context) error {
	branch := c.String("branch")
	repo := c.String("repo")
	base := c.String("base")
	remote := c.String("remote")
	directory := c.String("out")
	if repo == "" {
		return errors.New("repo must be defined if create-pr is true")
	}
	fmt.Printf("Creating a PR to repo %s with base %s and branch %s\n", repo, base, branch)
	r := &runner.CLIRunner{}
	g := git.NewCLIGit(git.CLIGitConfig{
		Directory: directory,
		Branch:    branch,
		Remote:    remote,
		Base:      base,
	}, r)
	scmClient, err := git.NewClient(git.SCMConfig{
		Branch: branch,
		Base:   base,
		Repo:   repo,
	})
	if err != nil {
		return fmt.Errorf("failed to create scm client: %w", err)
	}
	return catalog.CreatePullRequest(scmClient, g)
}
