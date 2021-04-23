package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"path/filepath"

	"github.com/weaveworks/pctl/pkg/client"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/writer"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/util/homedir"
)

func main() {
	app := &cli.App{
		Usage: "A cli tool for interacting with profiles",
		Flags: globalFlags(),
		Commands: []*cli.Command{
			searchCmd(),
			showCmd(),
			installCmd(),
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
				_ = cli.ShowCommandHelp(c, "show")
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
		},
		Action: func(c *cli.Context) error {
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
