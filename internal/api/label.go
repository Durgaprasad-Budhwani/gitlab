package api

import (
	"github.com/pinpt/agent/v4/sdk"
	"net/url"
)

type GitlabLabel struct {
	RefID int64 `json:"id"`
	Name string `json:"name"`
}

func ProjectLabelsPage(qc QueryContext, repo *GitlabProjectInternal, params url.Values) (page NextPage, lbls []*GitlabLabel, err error) {

	sdk.LogDebug(qc.Logger, "project labels", "repo", repo.Name, "repo_ref_id", repo.RefID, "params", params)

	objectPath := sdk.JoinURL("projects", url.QueryEscape(repo.RefID), "labels")

	page, err = qc.Get(objectPath, params, &lbls)
	if err != nil {
		return
	}

	sdk.LogDebug(qc.Logger, "labels", "v", lbls)

	return
}