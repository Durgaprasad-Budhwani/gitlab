package internal

import (
	"github.com/pinpt/gitlab/internal/common"
	"net/url"
	"strconv"
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

func (ge *GitlabExport2) exportPullRequestCommits(logger sdk.Logger, pr *internalPullRequest) error {

	var prCommits []*sdk.SourceCodePullRequestCommit

	repoRefID := strconv.FormatInt(*pr.repoRefID,10)
	repoID := sdk.NewSourceCodeRepoID(ge.customerID, repoRefID, common.GitlabRefType)
	pullRequestID := sdk.NewSourceCodePullRequestID(ge.customerID, strconv.FormatInt(pr.ID,10), common.GitlabRefType, repoID)

	err := api.Paginate(logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (api.NextPage, error) {
		np, commits, err := api.PullRequestCommitsPage2(logger, ge.qc, pr.repoRefID, pr.IID, params)
		if err != nil {
			return np, err
		}

		for _, commit := range commits {
			//if !after.IsZero() && rcommit.CreatedAt.Before(after) {
			//	return
			//}

			//author := commitAuthorUserToAuthor(&rcommit)
			//err = qc.UserManager.EmitGitUser(qc.Logger, author)
			//if err != nil {
			//	return
			//}
			//
			//author = commitCommiterUserToAuthor(&rcommit)
			//err = qc.UserManager.EmitGitUser(qc.Logger, author)
			//if err != nil {
			//	return
			//}

			item := commit.ToSourceCodePullRequestCommit(ge.customerID, common.GitlabRefType , repoID, pullRequestID)

			prCommits = append(prCommits, item)
		}

		return np, nil
	})

	if err != nil {
		return err
	}

	sdkPr := pr.ApiPullRequest.ToSourceCodePullRequest(logger, ge.customerID, repoID, common.GitlabRefType)

	setPullRequestCommits(sdkPr, prCommits)

	sdkPr.IntegrationInstanceID = ge.integrationInstanceID
	if err := ge.pipe.Write(sdkPr); err != nil {
		return err
	}

	for _, commit := range prCommits {
		commit.IntegrationInstanceID = ge.integrationInstanceID
		if err := ge.pipe.Write(commit); err != nil {
			return err
		}
	}



	return nil
}

//func (ge *GitlabExport2) fetchPullRequestsCommits(logger sdk.Logger, repoRefID *int64, prIID *int64) (commits []*sdk.SourceCodePullRequestCommit, rerr error) {
//
//
//	return
//
//}