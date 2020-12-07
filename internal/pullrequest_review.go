package internal

import (
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/gitlab/internal/common"

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
	_, review, err := api.PullRequestReviews2(logger, ge.qc, pr.repoRefID, pr.IID, nil)
	if err != nil {
		return err
	}

	repoRefID := strconv.FormatInt(*pr.repoRefID, 10)

	prIID := strconv.FormatInt(*pr.IID, 10)

	repoID := sdk.NewSourceCodeRepoID(ge.customerID, repoRefID, common.GitlabRefType)
	prID := sdk.NewSourceCodePullRequestID(ge.customerID, prIID, common.GitlabRefType, repoID)

	for _, a := range review.ApprovedBy {
		requestReview := &sdk.SourceCodePullRequestReview{}
		requestReview.CustomerID = ge.customerID
		requestReview.IntegrationInstanceID = ge.integrationInstanceID
		requestReview.RefType = common.GitlabRefType
		requestReview.RefID = strconv.FormatInt(review.ID, 10)
		requestReview.RepoID = repoID
		requestReview.PullRequestID = prID
		requestReview.Active = true
		requestReview.State = sdk.SourceCodePullRequestReviewStateApproved

		sdk.ConvertTimeToDateModel(review.CreatedAt, &requestReview.CreatedDate)

		requestReview.UserRefID = strconv.FormatInt(a.User.ID, 10)

		if err := ge.pipe.Write(requestReview); err != nil {
			return err
		}

	}

	for _, a := range review.SuggestedApprovers {

		requestReview := &sdk.SourceCodePullRequestReview{}
		requestReview.CustomerID = ge.customerID
		requestReview.IntegrationInstanceID = ge.integrationInstanceID
		requestReview.RefType = common.GitlabRefType
		requestReview.RefID = strconv.FormatInt(review.ID, 10)
		requestReview.RepoID = repoID
		requestReview.PullRequestID = prID
		requestReview.Active = true
		requestReview.State = sdk.SourceCodePullRequestReviewStateRequested
		sdk.ConvertTimeToDateModel(review.CreatedAt, &requestReview.CreatedDate)
		requestReview.UserRefID = strconv.FormatInt(a.UserID, 10)
		if err := ge.pipe.Write(requestReview); err != nil {
			return err
		}

		prAuthorRefID := strconv.FormatInt(pr.Author.ID, 10)

		reviewRequest := reviewRequest(ge.customerID, ge.integrationInstanceID, repoID, prID, requestReview.UserRefID, prAuthorRefID, pr.UpdatedAt)
		if err := ge.pipe.Write(&reviewRequest); err != nil {
			return err
		}

	}

	return nil
}

func reviewRequest(customerID string, integrationInstanceID *string, repoID string, prID string, requestedReviewerID string, senderRefID string, updatedAt time.Time) sdk.SourceCodePullRequestReviewRequest {

	review := sdk.SourceCodePullRequestReviewRequest{
		CustomerID:             customerID,
		ID:                     sdk.NewSourceCodePullRequestReviewRequestID(customerID, common.GitlabRefType, prID, requestedReviewerID),
		RefType:                common.GitlabRefType,
		RepoID:                 repoID,
		PullRequestID:          prID,
		Active:                 true,
		IntegrationInstanceID:  integrationInstanceID,
		RequestedReviewerRefID: requestedReviewerID,
		SenderRefID:            senderRefID,
	}

	sdk.ConvertTimeToDateModel(updatedAt, &review.CreatedDate)

	return review
}
