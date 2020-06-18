package internal

import (
	"net/url"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (g *GitlabIntegration) fetchPullRequestsCommits(repo *sdk.SourceCodeRepo, pr api.PullRequest) (commits []*sdk.SourceCodePullRequestCommit, rerr error) {
	rerr = api.PaginateStartAt(g.logger, "", func(log sdk.Logger, params url.Values) (api.PageInfo, error) {
		params.Set("per_page", MaxFetchedEntitiesCount)
		pi, commitsArr, err := api.PullRequestCommitsPage(g.qc, repo, pr, params)
		if err != nil {
			return pi, err
		}

		for _, c := range commitsArr {
			commits = append(commits, c)
		}
		return pi, nil
	})

	return

}

func (g *GitlabIntegration) exportPullRequestCommits(repo *sdk.SourceCodeRepo, pr api.PullRequest) (rerr error) {

	sdk.LogDebug(g.logger, "exporting pull requests commits", "pr", pr.Identifier)

	commits, err := g.fetchPullRequestsCommits(repo, pr)
	if err != nil {
		rerr = err
		return
	}

	setPullRequestCommits(pr.SourceCodePullRequest, commits)
	if err := g.writePullRequestCommits(commits); err != nil {
		rerr = err
		return
	}

	return
}
