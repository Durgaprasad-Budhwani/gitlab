package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

func (ge *GitlabExport) fetchPullRequestsCommits(repo *api.GitlabProjectInternal, pr api.PullRequest) (commits []*sdk.SourceCodePullRequestCommit, rerr error) {
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

func (ge *GitlabExport) FetchPullRequestsCommitsAfter(repo *api.GitlabProjectInternal, pr api.PullRequest, after time.Time) (commits []*sdk.SourceCodePullRequestCommit, rerr error) {
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

func (ge *GitlabExport) exportPullRequestCommits(repo *api.GitlabProjectInternal, pr api.PullRequest) error {

	sdk.LogDebug(ge.logger, "exporting pull requests commits", "pr", pr.Identifier)

	commits, err := ge.fetchPullRequestsCommits(repo, pr)
	if err != nil {
		return err
	}

	setPullRequestCommits(pr.SourceCodePullRequest, commits)
	if err := ge.writePullRequestCommits(commits); err != nil {
		return err
	}

	return nil
}
