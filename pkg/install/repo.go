package install

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/google/uuid"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

// GetProfileDefinition returns a definition based on a url and a branch.
func (i *Installer) GetProfileDefinition(repoURL, branch, path string) (profilesv1.ProfileDefinition, error) {
	// Add postfix so potential nested profiles don't clone into the same folder.
	u, err := uuid.NewRandom()
	if err != nil {
		return profilesv1.ProfileDefinition{}, err
	}
	// this should not be possible, but I don't like leaving open spots for an index overflow
	if len(u.String()) < 7 {
		return profilesv1.ProfileDefinition{}, errors.New("the generated uuid is not long enough")
	}

	var (
		tmp string
	)
	if v, ok := i.cloneCache[cloneCacheKey(repoURL, branch)]; ok {
		tmp = v
	} else {
		px := u.String()[:6]
		tmp, err = ioutil.TempDir("", "cloned_profile"+px)
		if err != nil {
			return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to create temp folder for cloning repository: %w", err)
		}
		if err := i.GitClient.Clone(repoURL, branch, tmp); err != nil {
			return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to clone the repo: %w", err)
		}
		i.cloneCache[cloneCacheKey(repoURL, branch)] = tmp
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
