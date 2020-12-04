package internal

import (
	"github.com/pinpt/gitlab/internal/common"
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

func (ge *GitlabExport) exportPullRequestsReviews(repo *api.GitlabProjectInternal, pr api.PullRequest) error {
	return api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (np api.NextPage, rerr error) {
		pi, reviews, err := api.PullRequestReviews(ge.qc, repo, pr, params)
		if err != nil {
			return pi, err
		}
		for _, c := range reviews {
			c.IntegrationInstanceID = ge.integrationInstanceID
			if err := ge.pipe.Write(c); err != nil {
				return
			}
		}
		return
	})
}

func (ge *GitlabExport2) exportPullRequestsReviews(logger sdk.Logger, pr *internalPullRequest) error {
	//return api.Paginate(logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (np api.NextPage, rerr error) {
		_, review, err := api.PullRequestReviews2(logger, ge.qc, pr.repoRefID, pr.IID, nil)
		if err != nil {
			return  err
		}

		repoRefID := strconv.FormatInt(*pr.repoRefID,10)

		prIID := strconv.FormatInt(*pr.IID,10)

		repoID := sdk.NewSourceCodeRepoID(ge.customerID, repoRefID, common.GitlabRefType)
		prID := sdk.NewSourceCodePullRequestID(ge.customerID, prIID, common.GitlabRefType, repoID)

		for _, a := range review.ApprovedBy {
			item := &sdk.SourceCodePullRequestReview{}
			item.CustomerID = ge.customerID
			item.RefType = common.GitlabRefType
			item.RefID = strconv.FormatInt(review.ID,10)
			item.RepoID = repoID
			item.PullRequestID = prID
			item.Active = true
			item.State = sdk.SourceCodePullRequestReviewStateApproved

			sdk.ConvertTimeToDateModel(review.CreatedAt, &item.CreatedDate)

			item.UserRefID = strconv.FormatInt(a.User.ID, 10)

			// TODO: check if pr.UpdateAt is fine here
			reviewRequest := reviewRequest(ge.customerID,ge.integrationInstanceID, repoID,prID, item.UserRefID, pr.UpdatedAt)
			reviewRequest.Active = false // TODO: I need to investigate this
			if err = ge.pipe.Write(&reviewRequest); err != nil{
				return err
			}

		}

		// TODO: add rreview.SuggestedApprovers

		return nil
	//})
}

func reviewRequest(customerID string, integrationInstanceID *string,repoID string,prID string, requestedReviewerID string, updatedAt time.Time) sdk.SourceCodePullRequestReviewRequest {

	review := sdk.SourceCodePullRequestReviewRequest{
		CustomerID:             customerID,
		ID:                     sdk.NewSourceCodePullRequestReviewRequestID(customerID, common.GitlabRefType, prID, requestedReviewerID),
		RefType:                common.GitlabRefType,
		RepoID:                 repoID,
		PullRequestID:          prID,
		Active:                 true,
		//CreatedDate:            sdk.SourceCodePullRequestReviewRequestCreatedDate(pr.UpdatedDate),
		IntegrationInstanceID:  integrationInstanceID,
		RequestedReviewerRefID: requestedReviewerID,
		//SenderRefID:            pr.CreatedByRefID, TODO: fix this
	}

	sdk.ConvertTimeToDateModel(updatedAt,&review.CreatedDate)

	return review
}