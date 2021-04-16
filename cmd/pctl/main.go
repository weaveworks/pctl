package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"

	"github.com/weaveworks/pctl/pkg/catalog"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	"github.com/weaveworks/profiles/pkg/git"
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
		UsageText: "pctl --catalog-url <URL> search <QUERY>",
		Action: func(c *cli.Context) error {
			searchName, catalogURL, err := parseArgs(c)
			if err != nil {
				_ = cli.ShowCommandHelp(c, "show")
				return err
			}

			fmt.Printf("searching for profiles matching %q:\n", searchName)
			profiles, err := catalog.Search(catalogURL, searchName)
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
		UsageText: "pctl --catalog-url <URL> show <CATALOG>/<PROFILE>",
		Action: func(c *cli.Context) error {
			profilePath, catalogURL, err := parseArgs(c)
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
			profile, err := catalog.Show(catalogURL, catalogName, profileName)
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
		Usage:     "generate configuration objects for later cluster setup",
		UsageText: "pctl --catalog-url <URL> install <CATALOG>/<PROFILE> --subscription-name pctl-profile --namespace default --branch main",
		Action: func(c *cli.Context) error {
			profilePath, catalogURL, err := parseArgs(c)
			if err != nil {
				_ = cli.ShowCommandHelp(c, "install")
				return err
			}

			branch := c.String("branch")
			if branch == "" {
				branch = "main"
			}
			subName := c.String("subscription-name")
			if subName == "" {
				subName = "pctl-profile"
			}
			namespace := c.String("namespace")
			if namespace == "" {
				namespace = "default"
			}

			parts := strings.Split(profilePath, "/")
			if len(parts) < 2 {
				_ = cli.ShowCommandHelp(c, "install")
				return errors.New("both catalog name and profile name must be provided")
			}
			catalogName, profileName := parts[0], parts[1]

			fmt.Printf("generating data for profile %s/%s:\n\n", catalogName, profileName)
			objs, err := catalog.Install(catalogURL, catalogName, profileName, subName, namespace, branch, git.GetProfileDefinition)
			if err != nil {
				return fmt.Errorf("failed to install yaml artifacts: %w", err)
			}
			return createFilesForArtifacts(objs)
		},
	}
}

func globalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "catalog-url",
			Usage:   "Catalog url",
			EnvVars: []string{"PCTL_CATALOG_URL"},
		},
	}
}

func parseArgs(c *cli.Context) (string, string, error) {
	catalogURL := c.String("catalog-url")
	if catalogURL == "" {
		return "", "", fmt.Errorf("--catalog-url or $PCTL_CATALOG_URL must be provided")
	}
	if c.Args().Len() < 1 {
		return "", "", fmt.Errorf("argument must be provided")
	}
	return c.Args().First(), catalogURL, nil
}

func printProfile(profile profilesv1.ProfileDescription) error {
	out, err := yaml.Marshal(profile)
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}

// createFilesForArtifacts will create files for all the artifacts that profiles generated.
func createFilesForArtifacts(artifacts []runtime.Object) error {
	e := kjson.NewSerializerWithOptions(kjson.DefaultMetaFactory, nil, nil, kjson.SerializerOptions{Yaml: true, Strict: true})
	generateOutput := func(i int, o runtime.Object) error {
		filename := fmt.Sprintf("%s_%d_resource.%s", o.GetObjectKind().GroupVersionKind().Kind, i, "yaml")
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		defer func(f *os.File) {
			if err := f.Close(); err != nil {
				fmt.Printf("Failed to properly close file %s\n", f.Name())
			}
		}(f)
		if err := e.Encode(o, f); err != nil {
			return err
		}
		return nil
	}
	for i, a := range artifacts {
		if err := generateOutput(i, a); err != nil {
			return err
		}
	}
	return nil
}
