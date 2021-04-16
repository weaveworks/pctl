package catalog

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	pp "github.com/weaveworks/profiles/pkg/profile"
)

// getProfileDefinition defines a client which can get profile definitions.
type getProfileDefinition func(repoURL, branch string, log logr.Logger) (profilesv1.ProfileDefinition, error)

// Install using the catalog at catalogURL and a profile matching the provided profileName generates all the
// artifacts and outputs a single yaml file containing all artifacts that the profile would create.
func Install(catalogURL, catalogName, profileName, subName, namespace, branch string, gitClient getProfileDefinition) ([]runtime.Object, error) {
	u, err := url.Parse(catalogURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url %q: %w", catalogURL, err)
	}

	u.Path = fmt.Sprintf("profiles/%s/%s", catalogName, profileName)
	resp, err := doRequest(u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close the response body from profile show with error: %v/n", err)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("unable to find profile `%s` in catalog `%s`", profileName, catalogName)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch profile: status code %d", resp.StatusCode)
	}

	profile := profilesv1.ProfileDescription{}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}
	log.Println(profile.URL)

	logger := logr.Discard()
	pd, err := gitClient(profile.URL, branch, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile definition: %w", err)
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
	return p.MakeOwnerlessArtifacts()
}
