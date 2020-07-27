package api

import (
	"net/url"
	"strconv"

	"github.com/pinpt/agent.next/sdk"
	pstrings "github.com/pinpt/go-common/v10/strings"
)

func PullRequestPage(
	qc QueryContext,
	repo *sdk.SourceCodeRepo,
	params url.Values,
	prs chan PullRequest) (pi NextPage, err error) {

	params.Set("scope", "all")
	params.Set("state", "all")

	sdk.LogDebug(qc.Logger, "repo pull requests", "repo", repo.Name, "repo_ref_id", repo.RefID, "params", params)

	objectPath := pstrings.JoinURL("projects", repo.RefID, "merge_requests")

	var rprs []apiPullRequest

	pi, err = qc.Get(objectPath, params, &rprs)
	if err != nil {
		return
	}

	repoID := sdk.NewSourceCodeRepoID(qc.CustomerID, repo.RefID, qc.RefType)

	for _, rpr := range rprs {
		pr := rpr.toSourceCodePullRequest(qc.Logger, qc.CustomerID, repoID, qc.RefType)

		spr := PullRequest{}
		spr.IID = strconv.FormatInt(rpr.IID, 10)
		spr.SourceCodePullRequest = pr
		prs <- spr
	}

	return
}
