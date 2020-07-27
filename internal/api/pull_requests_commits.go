package api

import (
	"net/url"
	"time"

	"github.com/pinpt/agent.next/sdk"
)

func PullRequestCommitsPage(
	qc QueryContext,
	repo *sdk.SourceCodeRepo,
	pr PullRequest,
	params url.Values) (pi NextPage, res []*sdk.SourceCodePullRequestCommit, err error) {

	sdk.LogDebug(qc.Logger, "pull request commits", "repo", repo.Name, "repo_ref_id", repo.RefID, "pr_iid", pr.IID, "params", params)

	objectPath := sdk.JoinURL("projects", repo.RefID, "merge_requests", pr.IID, "commits")

	var rcommits []struct {
		ID             string    `json:"id"`
		Message        string    `json:"message"`
		CreatedAt      time.Time `json:"created_at"`
		AuthorEmail    string    `json:"author_email"`
		CommitterEmail string    `json:"committer_email"`
		WebURL         string    `json:"web_url"`
	}

	pi, err = qc.Get(objectPath, params, &rcommits)
	if err != nil {
		return
	}

	repoID := sdk.NewSourceCodeRepoID(qc.CustomerID, repo.RefID, qc.RefType)
	pullRequestID := sdk.NewSourceCodePullRequestID(qc.CustomerID, pr.RefID, qc.RefType, repoID)

	for _, rcommit := range rcommits {

		item := &sdk.SourceCodePullRequestCommit{}
		item.CustomerID = qc.CustomerID
		item.RefType = qc.RefType
		item.RefID = rcommit.ID
		item.RepoID = repoID
		item.PullRequestID = pullRequestID
		item.Sha = rcommit.ID
		item.Message = rcommit.Message
		item.URL = rcommit.WebURL
		sdk.ConvertTimeToDateModel(rcommit.CreatedAt, &item.CreatedDate)

		item.AuthorRefID = CodeCommitEmail(qc.CustomerID, rcommit.AuthorEmail)
		item.CommitterRefID = CodeCommitEmail(qc.CustomerID, rcommit.CommitterEmail)

		res = append(res, item)
	}

	return
}

func CodeCommitEmail(customerID string, email string) string {
	sdk.Hash()
	return sdk.Hash(customerID, email)
}
