package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/client"
	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/runner"
)

func installCmd() *cli.Command {
	return &cli.Command{
		Name:  "install",
		Usage: "generate a profile subscription",
		UsageText: "To install from a profile catalog entry: pctl --catalog-url <URL> install --subscription-name pctl-profile --namespace default --profile-branch main --config-secret configmap-name <CATALOG>/<PROFILE>[/<VERSION>]\n   " +
			"To install directly from a profile repository: pctl install --subscription-name pctl-profile --namespace default --profile-branch development --profile-url https://github.com/weaveworks/profiles-examples --profile-path bitnami-nginx",
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
				Name:        "profile-branch",
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
				Name:        "pr-remote",
				Value:       "origin",
				DefaultText: "origin",
				Usage:       "The remote to push the branch to.",
			},
			&cli.StringFlag{
				Name:        "pr-base",
				Value:       "main",
				DefaultText: "main",
				Usage:       "The base branch to open a PR against.",
			},
			&cli.StringFlag{
				Name:  "pr-branch",
				Usage: "The branch to create the PR from. Generated if not set.",
			},
			&cli.StringFlag{
				Name:        "out",
				DefaultText: "current",
				Value:       ".",
				Usage:       "Optional location to create the profile installation folder in. This should be relative to the current working directory.",
			},
			&cli.StringFlag{
				Name:  "pr-repo",
				Value: "",
				Usage: "The repository to open a pr against. Format is: org/repo-name.",
			},
			&cli.StringFlag{
				Name:  "profile-url",
				Value: "",
				Usage: "Optional value defining the URL of the profile.",
			},
			&cli.StringFlag{
				Name:        "profile-path",
				Value:       ".",
				DefaultText: "<root>",
				Usage:       "Value defining the path to a profile when url is provided.",
			},
			&cli.StringFlag{
				Name:  "git-repository",
				Value: "",
				Usage: "The namespace and name of the GitRepository object governing the flux repo.",
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
	var (
		err           error
		catalogClient *client.Client
		profilePath   string
		catalogName   string
		profileName   string
		version       = "latest"
	)

	// only set up the catalog if a url is not provided
	url := c.String("profile-url")
	if url != "" && c.Args().Len() > 0 {
		return errors.New("it looks like you provided a url with a catalog entry; please choose either format: url/branch/path or <CATALOG>/<PROFILE>[/<VERSION>]")
	}

	if url == "" {
		profilePath, catalogClient, err = parseArgs(c)
		if err != nil {
			_ = cli.ShowCommandHelp(c, "install")
			return err
		}
		parts := strings.Split(profilePath, "/")
		if len(parts) < 2 {
			_ = cli.ShowCommandHelp(c, "install")
			return errors.New("both catalog name and profile name must be provided")
		}
		if len(parts) == 3 {
			version = parts[2]
		}
		catalogName, profileName = parts[0], parts[1]
	}

	branch := c.String("profile-branch")
	subName := c.String("subscription-name")
	namespace := c.String("namespace")
	configValues := c.String("config-secret")
	dir := c.String("out")
	path := c.String("profile-path")
	gitRepository := c.String("git-repository")

	var name string
	if url != "" {
		name = fmt.Sprintf("%s/%s", path, branch)
	} else {
		name = fmt.Sprintf("%s/%s", catalogName, profileName)
	}
	fmt.Printf("generating subscription for profile %s:\n\n", name)
	r := &runner.CLIRunner{}
	g := git.NewCLIGit(git.CLIGitConfig{}, r)
	cfg := catalog.InstallConfig{
		Clients: catalog.Clients{
			CatalogClient: catalogClient,
			GitClient:     g,
		},
		ProfileConfig: catalog.ProfileConfig{
			CatalogName:   catalogName,
			ConfigMap:     configValues,
			GitRepository: gitRepository,
			Namespace:     namespace,
			Path:          path,
			ProfileBranch: branch,
			ProfileName:   profileName,
			SubName:       subName,
			URL:           url,
			Version:       version,
		},
		Directory: dir,
	}
	return catalog.Install(cfg)
}

// createPullRequest runs the pull request creation part of the `install` command.
func createPullRequest(c *cli.Context) error {
	branch := c.String("pr-branch")
	repo := c.String("pr-repo")
	base := c.String("pr-base")
	remote := c.String("pr-remote")
	directory := c.String("out")
	if repo == "" {
		return errors.New("repo must be defined if create-pr is true")
	}
	if branch == "" {
		branch = c.String("subscription-name") + "-" + uuid.NewString()[:6]
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
