package internal

import (
	"net/url"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (g *GitlabIntegration) exportPullRequestsReviews(repo *sdk.SourceCodeRepo, pr *api.PullRequest) error {
	return api.PaginateStartAt(g.logger, "", func(log sdk.Logger, params url.Values) (pi api.PageInfo, rerr error) {
		params.Set("per_page", MaxFetchedEntitiesCount)
		pi, reviews, err := api.PullRequestReviewsPage(g.qc, repo, pr, params)
		if err != nil {
			return pi, err
		}
		for _, c := range reviews {
			if err := g.pipe.Write(c); err != nil {
				return
			}
		}
		return
	})
}
