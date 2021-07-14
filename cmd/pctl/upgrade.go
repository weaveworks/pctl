package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/runner"
	upgr "github.com/weaveworks/pctl/pkg/upgrade"
)

func upgradeCmd() *cli.Command {
	return &cli.Command{
		Name:      "upgrade",
		Usage:     "upgrade profile installation",
		UsageText: "To upgrade an installation: pctl upgrade pctl-profile-installation-path/ v0.1.1 ",
		Flags: append(createPRFlags, &cli.StringFlag{
			Name:  "git-repository",
			Value: "",
			Usage: "The namespace and name of the GitRepository object governing the flux repo.",
		}),
		Action: func(c *cli.Context) error {
			// Run upgradeation main
			if err := upgrade(c); err != nil {
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

func upgrade(c *cli.Context) error {
	if c.Args().Len() != 2 {
		return fmt.Errorf("please provid the path to the profile and version to upgrade to, e.g. pctl upgrade my-profile/ v0.1.1")
	}

	profilePath := c.Args().Slice()[0]
	profileVersion := c.Args().Slice()[1]
	catalogClient, err := getCatalogClient(c)
	if err != nil {
		return fmt.Errorf("failed to create catalog client: %w", err)
	}

	fmt.Printf("upgrading profile %q to version %q\n", profilePath, profileVersion)
	tmpDir, err := ioutil.TempDir("", "profile-upgrade")
	if err != nil {
		return err
	}
	defer func() {
		// os.RemoveAll(tmpDir)
		fmt.Println("would of deleted: ", tmpDir)
	}()

	var (
		gitRepoNamespace string
		gitRepoName      string
	)

	gitRepository := c.String("git-repository")
	if gitRepository != "" {
		split := strings.Split(gitRepository, "/")
		if len(split) != 2 {
			return fmt.Errorf("git-repository must in format <namespace>/<name>; was: %s", gitRepository)
		}
		gitRepoNamespace = split[0]
		gitRepoName = split[1]
	}

	cfg := upgr.UpgradeConfig{
		ProfileDir:     profilePath,
		Version:        profileVersion,
		CatalogClient:  catalogClient,
		CatalogManager: &catalog.Manager{},
		GitClient: git.NewCLIGit(git.CLIGitConfig{
			Directory: tmpDir,
		}, &runner.CLIRunner{}),
		GitRepoName:      gitRepoName,
		GitRepoNamespace: gitRepoNamespace,
	}
	err = upgr.Upgrade(cfg)
	if err == nil {
		fmt.Println("upgrade completed successfully")
	}
	return err
}
