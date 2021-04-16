package catalog

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	pg "github.com/weaveworks/profiles/pkg/git"
	pp "github.com/weaveworks/profiles/pkg/profile"
)

// Install using the catalog at catalogURL and a profile matching the provided profileName generates all the
// artifacts and outputs a single yaml file containing all artifacts that the profile would create.
func Install(catalogURL, catalogName, profileName, subName, namespace, branch string) error {
	u, err := url.Parse(catalogURL)
	if err != nil {
		return fmt.Errorf("failed to parse url %q: %w", catalogURL, err)
	}

	u.Path = fmt.Sprintf("profiles/%s/%s", catalogName, profileName)
	resp, err := doRequest(u, nil)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close the response body from profile show with error: %v/n", err)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("unable to find profile `%s` in catalog `%s`", profileName, catalogName)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch profile: status code %d", resp.StatusCode)
	}

	profile := profilesv1.ProfileDescription{}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return fmt.Errorf("failed to parse profile: %w", err)
	}
	log.Println(profile.URL)

	logger := logr.Discard()
	pd, err := pg.GetProfileDefinition(profile.URL, branch, logger)
	if err != nil {
		return fmt.Errorf("failed to get profile definition: %w", err)
	}

	p := pp.New(pd, profilesv1.ProfileSubscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      subName,
			Namespace: namespace,
		},
		Spec: profilesv1.ProfileSubscriptionSpec{
			ProfileURL: profile.URL,
			Branch:     branch,
		},
	}, nil, logger)
	objs, err := p.MakeOwnerlessArtifacts()
	if err != nil {
		return fmt.Errorf("failed to generate artifacts: %w", err)
	}
	return printArtifacts(objs)
}

func printArtifacts(artifacts []runtime.Object) error {
	generateOutput := func(t, name string, parse interface{}) error {
		content, err := ioutil.ReadFile(filepath.Join("crd_templates", t))
		if err != nil {
			return err
		}
		tmpl := template.Must(template.New("template").Parse(string(content)))
		base := strings.TrimSuffix(t, path.Ext(t))
		filename := fmt.Sprintf("%s_%s.%s", base, name, "yaml")
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		defer func(f *os.File) {
			if err := f.Close(); err != nil {
				fmt.Printf("Failed to properly close file %s\n", f.Name())
			}
		}(f)
		if err := tmpl.Execute(f, parse); err != nil {
			return err
		}
		return nil
	}
	//TODO: figure out a naming for these files.
	for _, a := range artifacts {
		switch t := a.(type) {
		case *sourcev1.GitRepository:
			if err := generateOutput("git_repository.tmpl", t.Name, a); err != nil {
				return err
			}
		case *sourcev1.HelmRepository:
			if err := generateOutput("helm_repository.tmpl", t.Name, a); err != nil {
				return err
			}
		case *helmv2.HelmRelease:
			if err := generateOutput("helm_release.tmpl", t.Name, a); err != nil {
				return err
			}
		case *kustomizev1.Kustomization:
			if err := generateOutput("kustomization.tmpl", t.Name, a); err != nil {
				return err
			}
		}
	}
	return nil
}
