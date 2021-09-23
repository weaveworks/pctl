package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/weaveworks/kivo-cli/pkg/catalog"
	"github.com/weaveworks/kivo-cli/pkg/git"
	"github.com/weaveworks/kivo-cli/pkg/log"
	"github.com/weaveworks/kivo-cli/pkg/runner"
	upgr "github.com/weaveworks/kivo-cli/pkg/upgrade"
	"github.com/weaveworks/kivo-cli/pkg/upgrade/repo"
)

func upgradeCmd() *cli.Command {
	return &cli.Command{
		Name:      "upgrade",
		Usage:     "upgrade profile installation",
		UsageText: "To upgrade an installation: kivo upgrade kivo-profile-installation-path/ v0.1.1",
		Flags: append(createPRFlags, &cli.BoolFlag{
			Name:        "latest",
			Usage:       "--latest",
			DefaultText: "*WARNING*: Upgrade to the latest version. Please ensure this is a valid upgrade path before proceeding",
		}),
		Action: func(c *cli.Context) error {
			if err := upgrade(c); err != nil {
				return err
			}
			if c.Bool("create-pr") {
				profilePath := c.Args().Slice()[0]
				if err := createPullRequest(c, profilePath); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func upgrade(c *cli.Context) error {
	var (
		profilePath, profileVersion string
	)

	latest := c.Bool("latest")
	if latest {
		if c.Args().Len() != 1 {
			return fmt.Errorf("please provid the path to the profile to upgrade to, e.g. kivo upgrade my-profile/")
		}
		profilePath = c.Args().Slice()[0]
	} else {
		if c.Args().Len() != 2 {
			return fmt.Errorf("please provid the path to the profile and version to upgrade to, e.g. kivo upgrade my-profile/ v0.1.1")
		}
		profilePath = c.Args().Slice()[0]
		profileVersion = c.Args().Slice()[1]
	}

	catalogClient, err := getCatalogClient(c)
	if err != nil {
		return fmt.Errorf("failed to create catalog client: %w", err)
	}

	tmpDir, err := ioutil.TempDir("", "profile-upgrade")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Warningf("failed to cleanup temp directory %q: %v", tmpDir, err)
		}
	}()
	message := c.String("pr-message")
	cfg := upgr.Config{
		ProfileDir:     profilePath,
		Version:        profileVersion,
		CatalogClient:  catalogClient,
		CatalogManager: &catalog.Manager{},
		RepoManager: repo.NewManager(git.NewCLIGit(git.CLIGitConfig{
			Directory: tmpDir,
			Quiet:     true,
			Message:   message,
		}, &runner.CLIRunner{})),
		WorkingDir: tmpDir,
		Message:    message,
		Latest:     latest,
	}
	return upgr.Upgrade(cfg)
}
