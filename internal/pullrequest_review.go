package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (g *GitlabIntegration) exportPullRequestsReviews(repo *sdk.SourceCodeRepo, pr api.PullRequest) error {
	return api.Paginate(g.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (np api.NextPage, rerr error) {
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
