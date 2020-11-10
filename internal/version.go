package internal

import (
	"github.com/blang/semver"
	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

const minimumGitlabServerVersionSupported = "13.3.0-ee"

// ValidServerVersion valid server version
func (ge *GitlabExport) ValidServerVersion() (bool, error) {

	version, err := api.ServerVersion(ge.qc)
	if err != nil {
		return false, err
	}

	minimumSupportedVersion, err := semver.New(minimumGitlabServerVersionSupported)
	if err != nil {
		return false, err
	}
	currentVersion, err := semver.New(version)
	if err != nil {
		return false, err
	}

	sdk.LogDebug(ge.logger, "current server version", "current-version", version, "minimum-version", minimumGitlabServerVersionSupported)

	return currentVersion.GTE(*minimumSupportedVersion), nil
}
