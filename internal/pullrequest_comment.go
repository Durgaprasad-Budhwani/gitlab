package internal

import (
	"github.com/pinpt/gitlab/internal/common"
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

func (ge *GitlabExport) exportPullRequestsComments(repo *api.GitlabProjectInternal, pr api.PullRequest) error {
	return api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, rerr error) {
		pi, comments, err := api.PullRequestCommentsPage(ge.qc, repo, pr, params)
		if err != nil {
			return pi, err
		}
		for _, c := range comments {
			c.IntegrationInstanceID = ge.integrationInstanceID
			if err := ge.pipe.Write(c); err != nil {
				return
			}
		}
		return
	})
}

func (ge *GitlabExport2) exportPullRequestsComments(logger sdk.Logger,pr *internalPullRequest) (string, error) {

	u, err := url.Parse(ge.baseURL)
	if err != nil {
		return "1", err
	}

	repoRefID := strconv.FormatInt(*pr.repoRefID,10)
	repoID := sdk.NewSourceCodeRepoID(ge.customerID, repoRefID, common.GitlabRefType)
	prRefID := strconv.FormatInt(pr.ID,10)
	pullRequestID := sdk.NewSourceCodePullRequestID(ge.customerID, prRefID, common.GitlabRefType, repoID)
	prIID := strconv.FormatInt(*pr.IID,10)

	var lastPage api.NextPage

	err = api.Paginate2(api.NextPage(pr.page), false, time.Time{}, func(params url.Values, t time.Time) (api.NextPage, error) {
		np, comments, err := api.PullRequestCommentsPage2(logger, ge.qc, pr.repoRefID, pr.IID, params)
		lastPage = np
		if err != nil {
			return np, err
		}
		for _, comment := range comments {
			if comment.System {

				prReview := &sdk.SourceCodePullRequestReview{}
				prReview.CustomerID = ge.customerID
				prReview.IntegrationInstanceID = ge.integrationInstanceID
				prReview.Active = true
				prReview.CustomerID = ge.customerID
				prReview.RefType = common.GitlabRefType

				prReview.RepoID = repoID
				prReview.PullRequestID = pullRequestID

				prReview.RefID = strconv.FormatInt(comment.Author.ID,10)

				switch comment.Body {
					case "approved this merge request":
						prReview.State = sdk.SourceCodePullRequestReviewStateApproved
					case "unapproved this merge request":
						prReview.State = sdk.SourceCodePullRequestReviewStateDismissed
				}

				if err := ge.pipe.Write(prReview); err != nil {
					return "", err
				}

				continue
			}

			prComment := &sdk.SourceCodePullRequestComment{}
			prComment.Active = true
			prComment.CustomerID = ge.customerID
			prComment.IntegrationInstanceID = ge.integrationInstanceID
			prComment.RefType = common.GitlabRefType
			prComment.RefID = strconv.FormatInt(comment.ID,10)
			prComment.URL = sdk.JoinURL(u.Scheme, "://", u.Hostname(), *pr.repoFullName, "merge_requests",prIID)
			sdk.ConvertTimeToDateModel(comment.UpdatedAt, &prComment.UpdatedDate)

			prComment.RepoID = repoID
			prComment.PullRequestID = pullRequestID
			prComment.Body = comment.Body
			sdk.ConvertTimeToDateModel(comment.CreatedAt, &prComment.CreatedDate)

			prComment.UserRefID = strconv.FormatInt(comment.Author.ID, 10)

			if err := ge.pipe.Write(prComment); err != nil {
				return "",err
			}

			if comment.Type == "DiffNote" {
				prReview := &sdk.SourceCodePullRequestReview{}
				prReview.Active = true
				prReview.CustomerID = ge.customerID
				prReview.RefType = common.GitlabRefType
				prReview.RefID = strconv.FormatInt(comment.ID,10)
				prReview.IntegrationInstanceID = ge.integrationInstanceID

				prReview.RepoID = repoID
				prReview.PullRequestID = pullRequestID
				prReview.State = sdk.SourceCodePullRequestReviewStateCommented

				if err := ge.pipe.Write(prReview); err != nil {
					return "", err
				}
			}
		}

		return np, nil
	})
	if err != nil {
		return string(lastPage), err
	}

	return string(lastPage), nil
}
