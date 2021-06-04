package repo

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/google/uuid"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/git"
)

// GetProfileDefinition returns a definition based on a url and a branch.
func GetProfileDefinition(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error) {
	// Add postfix so potential nested profiles don't clone into the same folder.
	u, err := uuid.NewRandom()
	if err != nil {
		return profilesv1.ProfileDefinition{}, err
	}
	// this should not be possible, but I don't like leaving open spots for an index overflow
	if len(u.String()) < 7 {
		return profilesv1.ProfileDefinition{}, errors.New("the generated uuid is not long enough")
	}
	px := u.String()[:6]
	tmp, err := ioutil.TempDir("", "get_profile_definition_clone_"+px)
	if err != nil {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to create temp folder for cloning repository: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmp); err != nil {
			fmt.Println("failed to remove temp folder, please clean by hand: ", tmp)
		}
	}()

	if err := gitClient.Clone(repoURL, branch, tmp); err != nil {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to clone the repo: %w", err)
	}

	content, err := ioutil.ReadFile(filepath.Join(tmp, path, "profile.yaml"))
	if err != nil {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("could not find file at cloned location: %w", err)
	}

	profile := profilesv1.ProfileDefinition{}
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 4096).Decode(&profile)
	if err != nil {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to parse profile: %w", err)
	}

	return profile, nil
}
