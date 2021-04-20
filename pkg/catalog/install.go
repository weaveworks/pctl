package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
)

// Writer takes a profile and writes it out into a medium.
type Writer interface {
	Output(prof *profilesv1.ProfileSubscription) error
}

// FileWriter is a Writer using a file as backing medium.
type FileWriter struct {
	Filename string
}

// Output writes the profile subscription yaml data into a given file.
func (fw *FileWriter) Output(prof *profilesv1.ProfileSubscription) error {
	e := kjson.NewSerializerWithOptions(kjson.DefaultMetaFactory, nil, nil, kjson.SerializerOptions{Yaml: true, Strict: true})
	f, err := os.OpenFile(fw.Filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			fmt.Printf("Failed to close file %s\n", f.Name())
		}
	}(f)
	if err := e.Encode(prof, f); err != nil {
		return err
	}
	return nil
}

// StdoutWriter is a Writer using stdout as backing medium.
type StdoutWriter struct{}

// Output outputs the yaml generated content for a profile to stdout.
func (s *StdoutWriter) Output(prof *profilesv1.ProfileSubscription) error {
	e := kjson.NewSerializerWithOptions(kjson.DefaultMetaFactory, nil, nil, kjson.SerializerOptions{Yaml: true, Strict: true})
	return e.Encode(prof, os.Stdout)
}

// Install using the catalog at catalogURL and a profile matching the provided profileName generates all the
// artifacts and outputs a single yaml file containing all artifacts that the profile would create.
func Install(catalogURL, catalogName, profileName, subName, namespace, branch, configMap string, writer Writer) error {
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

	subscription := profilesv1.ProfileSubscription{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ProfileSubscription",
			APIVersion: "weave.works/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      subName,
			Namespace: namespace,
		},
		Spec: profilesv1.ProfileSubscriptionSpec{
			ProfileURL: profile.URL,
			Branch:     branch,
		},
	}
	if configMap != "" {
		subscription.Spec.ValuesFrom = []helmv2.ValuesReference{
			{
				Kind:      "ConfigMap",
				Name:      subName + "-values",
				ValuesKey: configMap,
			},
		}
	}
	if err := writer.Output(&subscription); err != nil {
		return fmt.Errorf("failed to output subscription information: %w", err)
	}
	return nil
}
