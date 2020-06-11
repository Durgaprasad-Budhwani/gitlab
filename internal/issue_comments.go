package internal

import (
	"net/url"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (g *GitlabIntegration) exportIssueComments(repo *sdk.SourceCodeRepo, pr *api.PullRequest) error {
	return api.PaginateStartAt(g.logger, "", func(log sdk.Logger, params url.Values) (pi api.PageInfo, rerr error) {
		pi, comments, err := api.PullRequestCommentsPage(g.qc, repo, pr, params)
		if err != nil {
			return pi, err
		}
		for _, c := range comments {
			if err := g.pipe.Write(c); err != nil {
				return
			}
		}
		return
	})
}
