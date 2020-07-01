package api

import (
	"fmt"
	"net/url"
	"time"

	"github.com/pinpt/agent.next/sdk"
	"github.com/pinpt/go-common/v10/datetime"
	pstrings "github.com/pinpt/go-common/v10/strings"
)

func PullRequestReviewsPage(
	qc QueryContext,
	repo *sdk.SourceCodeRepo,
	pr PullRequest,
	params url.Values) (pi PageInfo, res []*sdk.SourceCodePullRequestReview, err error) {

	sdk.LogDebug(qc.Logger, "pull request reviews", "repo", repo.Name, "repo_ref_id", repo.RefID, "pr_iid", pr.IID, "params", params)

	objectPath := pstrings.JoinURL("projects", repo.RefID, "merge_requests", pr.IID, "approvals")

	var rreview struct {
		ID         int64 `json:"id"`
		ApprovedBy []struct {
			User struct {
				Username string `json:"username"`
			} `json:"user"`
		} `json:"approved_by"`
		SuggestedApprovers []struct {
			User struct {
				Username string `json:"username"`
			} `json:"user"`
		} `json:"suggested_approvers"`
		Approvers []struct {
			User struct {
				Username string `json:"username"`
			} `json:"user"`
		} `json:"approvers"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	pi, err = qc.Request(objectPath, params, &rreview)
	if err != nil {
		return
	}

	repoID := sdk.NewSourceCodeRepoID(qc.CustomerID, repo.RefID, qc.RefType)
	pullRequestID := sdk.NewSourceCodePullRequestID(qc.CustomerID, pr.RefID, qc.RefType, repoID)

	for _, a := range rreview.ApprovedBy {
		item := &sdk.SourceCodePullRequestReview{}
		item.CustomerID = qc.CustomerID
		item.RefType = qc.RefType
		item.RefID = fmt.Sprint(rreview.ID)
		item.RepoID = repoID
		item.PullRequestID = pullRequestID
		item.State = sdk.SourceCodePullRequestReviewStateApproved

		datetime.ConvertToModel(rreview.CreatedAt, &item.CreatedDate)

		item.UserRefID = a.User.Username

		res = append(res, item)
	}

	for _, a := range rreview.SuggestedApprovers {
		item := &sdk.SourceCodePullRequestReview{}
		item.CustomerID = qc.CustomerID
		item.RefType = qc.RefType
		item.RefID = fmt.Sprint(rreview.ID)
		item.RepoID = repoID
		item.PullRequestID = pullRequestID
		item.State = sdk.SourceCodePullRequestReviewStatePending

		datetime.ConvertToModel(rreview.CreatedAt, &item.CreatedDate)

		item.UserRefID = a.User.Username

		res = append(res, item)
	}

	return
}
