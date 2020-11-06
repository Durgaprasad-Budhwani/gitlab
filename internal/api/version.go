package api

import (
	"github.com/pinpt/agent/v4/sdk"
)

type serverVersion struct {
	Version  string `json:"version"`
	Revision string `json:"revision"`
}

func ServerVersion(qc QueryContext) (s string, err error) {

	sdk.LogDebug(qc.Logger, "server version")

	objectPath := sdk.JoinURL("version")

	var sv serverVersion

	_, err = qc.Get(objectPath, nil, &sv)
	if err != nil {
		return "", err
	}

	sdk.LogDebug(qc.Logger, "current server version", "version", sv.Version, "revision", sv.Revision)

	return sv.Version, nil
}
