package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (ge *GitlabExport) fetchPullRequestsCommits(repo *sdk.SourceCodeRepo, pr api.PullRequest) (commits []*sdk.SourceCodePullRequestCommit, rerr error) {
	rerr = api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (api.NextPage, error) {
		pi, commitsArr, err := api.PullRequestCommitsPage(ge.qc, repo, pr, params, t)
		if err != nil {
			return pi, err
		}
		commits = append(commits, commitsArr...)
		return pi, nil
	})

	return

}

func (ge *GitlabExport) FetchPullRequestsCommitsAfter(repo *sdk.SourceCodeRepo, pr api.PullRequest, after time.Time) (commits []*sdk.SourceCodePullRequestCommit, rerr error) {
	rerr = api.Paginate(ge.logger, "", after, func(log sdk.Logger, params url.Values, t time.Time) (api.NextPage, error) {
		pi, commitsArr, err := api.PullRequestCommitsPage(ge.qc, repo, pr, params, t)
		if err != nil {
			return pi, err
		}

		commits = append(commits, commitsArr...)
		return pi, nil
	})

	return

}

func (ge *GitlabExport) exportPullRequestCommits(repo *sdk.SourceCodeRepo, pr api.PullRequest) (rerr error) {

	sdk.LogDebug(ge.logger, "exporting pull requests commits", "pr", pr.Identifier)

	commits, err := ge.fetchPullRequestsCommits(repo, pr)
	if err != nil {
		rerr = err
		return
	}

	setPullRequestCommits(pr.SourceCodePullRequest, commits)
	if err := ge.writePullRequestCommits(commits); err != nil {
		rerr = err
		return
	}

	return
}
