package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/gitlab/internal/api"
	"github.com/pinpt/agent/v4/sdk"
)

func (ge *GitlabExport) exportPullRequestsReviews(repo *sdk.SourceCodeRepo, pr api.PullRequest) error {
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
