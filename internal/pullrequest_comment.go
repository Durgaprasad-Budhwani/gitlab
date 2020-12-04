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

func (ge *GitlabExport2) exportPullRequestsComments(logger sdk.Logger,pr *internalPullRequest) error {

	u, err := url.Parse(ge.baseURL)
	if err != nil {
		return err
	}

	repoRefID := strconv.FormatInt(*pr.repoRefID,10)
	repoID := sdk.NewSourceCodeRepoID(ge.customerID, repoRefID, common.GitlabRefType)
	prRefID := strconv.FormatInt(pr.ID,10)
	pullRequestID := sdk.NewSourceCodePullRequestID(ge.customerID, prRefID, common.GitlabRefType, repoID)
	prIID := strconv.FormatInt(*pr.IID,10)

	return api.Paginate2("", false, time.Time{}, func(params url.Values, t time.Time) (pi api.NextPage, rerr error) {
		pi, comments, err := api.PullRequestCommentsPage2(logger, ge.qc, pr.repoRefID, pr.IID, params)
		if err != nil {
			return pi, err
		}
		for _, rcomment := range comments {
			if rcomment.System {
				continue
			}
			item := &sdk.SourceCodePullRequestComment{}
			item.Active = true
			item.CustomerID = ge.customerID
			item.RefType = common.GitlabRefType
			item.RefID = strconv.FormatInt(rcomment.ID,10)
			item.URL = sdk.JoinURL(u.Scheme, "://", u.Hostname(), *pr.repoFullName, "merge_requests",prIID)
			sdk.ConvertTimeToDateModel(rcomment.UpdatedAt, &item.UpdatedDate)

			item.RepoID = repoID
			item.PullRequestID = pullRequestID
			item.Body = rcomment.Body
			sdk.ConvertTimeToDateModel(rcomment.CreatedAt, &item.CreatedDate)

			item.UserRefID = strconv.FormatInt(rcomment.Author.ID, 10)
			item.IntegrationInstanceID = ge.integrationInstanceID
			if err := ge.pipe.Write(item); err != nil {
				return
			}
		}

		return
	})
}
