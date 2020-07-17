package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (ge *GitlabExport) exportIssueComments(repo *sdk.SourceCodeRepo, pr api.PullRequest) error {
	return api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, stopOnUpdatedAt time.Time) (pi api.NextPage, rerr error) {
		pi, comments, err := api.PullRequestCommentsPage(ge.qc, repo, pr, params)
		if err != nil {
			return pi, err
		}
		for _, c := range comments {
			if err := ge.pipe.Write(c); err != nil {
				return
			}
		}
		return
	})
}
