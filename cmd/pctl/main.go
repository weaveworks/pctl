package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/util/homedir"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/client"
	"github.com/weaveworks/pctl/pkg/cluster"
	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/runner"
	"github.com/weaveworks/pctl/pkg/writer"
)

const (
	releaseUrl = "https://github.com/weaveworks/profiles/releases"
)

func main() {
	app := &cli.App{
		Usage: "A cli tool for interacting with profiles",
		Flags: globalFlags(),
		Commands: []*cli.Command{
			searchCmd(),
			showCmd(),
			installCmd(),
			prepareCmd(),
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func searchCmd() *cli.Command {
	return &cli.Command{
		Name:      "search",
		Usage:     "search for a profile",
		UsageText: "pctl --kubeconfig=<kubeconfig-path> search <QUERY>",
		Action: func(c *cli.Context) error {
			searchName, catalogClient, err := parseArgs(c)
			if err != nil {
				_ = cli.ShowCommandHelp(c, "search")
				return err
			}

			fmt.Printf("searching for profiles matching %q:\n", searchName)
			profiles, err := catalog.Search(catalogClient, searchName)
			if err != nil {
				return err
			}
			for _, profile := range profiles {
				fmt.Printf("%s: %s\n", profile.Name, profile.Description)
			}
			return nil
		},
	}
}

func showCmd() *cli.Command {
	return &cli.Command{
		Name:      "show",
		Usage:     "display information about a profile",
		UsageText: "pctl --kubeconfig=<kubeconfig-path> show <CATALOG>/<PROFILE>",
		Action: func(c *cli.Context) error {
			profilePath, catalogClient, err := parseArgs(c)
			if err != nil {
				_ = cli.ShowCommandHelp(c, "show")
				return err
			}

			parts := strings.Split(profilePath, "/")
			if len(parts) < 2 {
				_ = cli.ShowCommandHelp(c, "show")
				return errors.New("both catalog name and profile name must be provided")
			}
			catalogName, profileName := parts[0], parts[1]

			fmt.Printf("retrieving information for profile %s/%s:\n\n", catalogName, profileName)
			profile, err := catalog.Show(catalogClient, catalogName, profileName)
			if err != nil {
				return err
			}
			return printProfile(profile)
		},
	}
}

func installCmd() *cli.Command {
	return &cli.Command{
		Name:      "install",
		Usage:     "generate a profile subscription for a profile in a catalog",
		UsageText: "pctl --catalog-url <URL> install --subscription-name pctl-profile --namespace default --branch main --config-secret configmap-name --out profile_subscription.yaml <CATALOG>/<PROFILE>",
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
			&cli.StringFlag{
				Name:        "out",
				Value:       "profile_subscription.yaml",
				DefaultText: "profile_subscription.yaml",
				Usage:       "Filename to use for the generated content.",
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
	filename := c.String("out")

	parts := strings.Split(profilePath, "/")
	if len(parts) < 2 {
		_ = cli.ShowCommandHelp(c, "install")
		return errors.New("both catalog name and profile name must be provided")
	}
	catalogName, profileName := parts[0], parts[1]

	fmt.Printf("generating subscription for profile %s/%s:\n\n", catalogName, profileName)
	w := &writer.FileWriter{Filename: filename}
	cfg := catalog.InstallConfig{
		Branch:        branch,
		CatalogName:   catalogName,
		CatalogClient: catalogClient,
		ConfigMap:     configValues,
		Namespace:     namespace,
		ProfileName:   profileName,
		SubName:       subName,
		Writer:        w,
	}
	return catalog.Install(cfg)
}

// createPullRequest runs the pull request creation part of the `install` command.
func createPullRequest(c *cli.Context) error {
	branch := c.String("branch")
	filename := c.String("out")
	repo := c.String("repo")
	base := c.String("base")
	remote := c.String("remote")
	if repo == "" {
		return errors.New("repo must be defined if create-pr is true")
	}
	fmt.Printf("Creating a PR to repo %s with base %s and branch %s\n", repo, base, branch)
	r := &runner.CLIRunner{}
	g := git.NewCLIGit(git.CLIGitConfig{
		Filename: filename,
		Location: filepath.Dir(filename),
		Branch:   branch,
		Remote:   remote,
		Base:     base,
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
				Version:     c.String("version"),
				KubeConfig:  c.String("kubeconfig"),
				KubeContext: c.String("context"),
				DryRun:      c.Bool("dry-run"),
			})
			if err != nil {
				return err
			}
			return p.Prepare()
		},
	}
}

func globalFlags() []cli.Flag {
	var kubeconfigFlag *cli.StringFlag
	if home := homedir.HomeDir(); home != "" {
		kubeconfigFlag = &cli.StringFlag{
			Name:  "kubeconfig",
			Value: filepath.Join(home, ".kube", "config"),
			Usage: "Absolute path to the kubeconfig file (optional)",
		}
	} else {
		kubeconfigFlag = &cli.StringFlag{
			Name:     "kubeconfig",
			Usage:    "Absolute path to the kubeconfig file",
			Required: true,
		}
	}

	return []cli.Flag{
		&cli.StringFlag{
			Name:  "catalog-service-name",
			Value: "profiles-catalog-service",
			Usage: "Catalog Kubernetes Service name",
		},
		&cli.StringFlag{
			Name:  "catalog-service-port",
			Value: "8000",
			Usage: "Catalog Kubernetes Service port",
		},
		&cli.StringFlag{
			Name:  "catalog-service-namespace",
			Value: "profiles-system",
			Usage: "Catalog Kubernetes Service namespace",
		},
		kubeconfigFlag,
	}
}

func parseArgs(c *cli.Context) (string, *client.Client, error) {
	options := client.ServiceOptions{
		KubeconfigPath: c.String("kubeconfig"),
		Namespace:      c.String("catalog-service-namespace"),
		ServiceName:    c.String("catalog-service-name"),
		ServicePort:    c.String("catalog-service-port"),
	}

	if c.Args().Len() < 1 {
		return "", nil, fmt.Errorf("argument must be provided")
	}
	client, err := client.NewFromOptions(options)
	if err != nil {
		return "", nil, err
	}
	return c.Args().First(), client, nil
}

func printProfile(profile profilesv1.ProfileDescription) error {
	out, err := yaml.Marshal(profile)
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}
