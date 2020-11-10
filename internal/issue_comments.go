package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

func (ge *GitlabExport) exportIssueComments(repo *api.GitlabProjectInternal, pr api.PullRequest) error {
	return api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, stopOnUpdatedAt time.Time) (pi api.NextPage, rerr error) {
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
