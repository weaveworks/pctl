package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/log"
	"github.com/weaveworks/pctl/pkg/runner"
	upgr "github.com/weaveworks/pctl/pkg/upgrade"
	"github.com/weaveworks/pctl/pkg/upgrade/repo"
)

func upgradeCmd() *cli.Command {
	return &cli.Command{
		Name:      "upgrade",
		Usage:     "upgrade profile installation",
		UsageText: "To upgrade an installation: pctl upgrade pctl-profile-installation-path/ v0.1.1 ",
		Flags:     createPRFlags,
		Action: func(c *cli.Context) error {
			if err := upgrade(c); err != nil {
				return err
			}
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
	cfg := upgr.UpgradeConfig{
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
	}
	return upgr.Upgrade(cfg)
}
