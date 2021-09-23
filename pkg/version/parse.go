package version

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	fluxcd "github.com/fluxcd/pkg/version"
)

// ParseKivoVersion parses the kivo version as semver while ignoring
// extra build metadata
func ParseKivoVersion(raw string) (*semver.Version, error) {
	// We don't want any extra info from the version
	semverVersion := strings.Split(raw, ExtraSep)[0]
	v, err := fluxcd.ParseVersion(semverVersion)
	if err != nil {
		return v, fmt.Errorf("unexpected error parsing kivo version %q", raw)
	}
	return v, nil
}
