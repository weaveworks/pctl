package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"github.com/weaveworks/pctl/pkg/client"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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
			listCmd(),
			prepareCmd(),
			docgenCmd(),
			upgradeCmd(),
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		// to prevent the timestamp in the output from log.Fatal.
		fmt.Println(err)
		os.Exit(1)
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
	if c.Args().Len() < 1 {
		return "", nil, fmt.Errorf("argument must be provided")
	}
	client, err := buildCatalogClient(c)
	if err != nil {
		return "", nil, err
	}
	return c.Args().First(), client, nil
}

func getCatalogClient(c *cli.Context) (*client.Client, error) {
	return buildCatalogClient(c)
}

func buildCatalogClient(c *cli.Context) (*client.Client, error) {
	options := client.ServiceOptions{
		KubeconfigPath: c.String("kubeconfig"),
		Namespace:      c.String("catalog-service-namespace"),
		ServiceName:    c.String("catalog-service-name"),
		ServicePort:    c.String("catalog-service-port"),
	}
	return client.NewFromOptions(options)
}

func buildK8sClient(kubeconfig string) (runtimeclient.Client, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create config from kubeconfig path %q: %w", kubeconfig, err)
	}
	cl, err := runtimeclient.New(config, runtimeclient.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	utilruntime.Must(profilesv1.AddToScheme(cl.Scheme()))
	return cl, nil
}
