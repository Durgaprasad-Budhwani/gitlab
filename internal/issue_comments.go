package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (g *GitlabIntegration) exportIssueComments(repo *sdk.SourceCodeRepo, pr *api.PullRequest) error {
	return api.Paginate(g.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, stopOnUpdatedAt time.Time) (pi api.NextPage, rerr error) {
		// TODO: Add concurrency to WORK type
		// pi, comments, err := api.PullRequestCommentsPage(g.qc, repo, pr, params)
		// if err != nil {
		// 	return pi, err
		// }
		// for _, c := range comments {
		// 	if err := g.pipe.Write(c); err != nil {
		// 		return
		// 	}
		// }
		return
	})
}
