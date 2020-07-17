package api

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/agent.next/sdk"
	"github.com/pinpt/go-common/v10/datetime"
	pstrings "github.com/pinpt/go-common/v10/strings"
)

func PullRequestReviewsPage(
	qc QueryContext,
	repo *sdk.SourceCodeRepo,
	pr PullRequest,
	params url.Values) (pi NextPage, res []*sdk.SourceCodePullRequestReview, err error) {

	sdk.LogDebug(qc.Logger, "pull request reviews", "repo", repo.Name, "repo_ref_id", repo.RefID, "pr_iid", pr.IID, "params", params)

	objectPath := pstrings.JoinURL("projects", repo.RefID, "merge_requests", pr.IID, "approvals")

	var rreview struct {
		ID         int64 `json:"id"`
		ApprovedBy []struct {
			User struct {
				ID int64 `json:"id"`
			} `json:"user"`
		} `json:"approved_by"`
		SuggestedApprovers []struct {
			User struct {
				ID int64 `json:"id"`
			} `json:"user"`
		} `json:"suggested_approvers"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	pi, err = qc.Get(objectPath, params, &rreview)
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

		item.UserRefID = strconv.FormatInt(a.User.ID, 10)

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

		item.UserRefID = strconv.FormatInt(a.User.ID, 10)

		res = append(res, item)
	}

	return
}
