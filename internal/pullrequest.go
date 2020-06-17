package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

// PullRequestFuture pull requests will process later
type PullRequestFuture struct {
	Repo *sdk.SourceCodeRepo
	Page api.PageInfo
}

func (g *GitlabIntegration) exportRepoPullRequests(repo *sdk.SourceCodeRepo) error {

	sdk.LogDebug(g.logger, "pull requests", "repo", repo.Name)

	page, prs, err := g.fetchInitialRepoPullRequests(repo)
	if err != nil {
		return err
	}

	g.addPullRequestFuture(repo, page)

	return g.exportPullRequestEntitiesAndWrite(repo, prs)
}

func (g *GitlabIntegration) addPullRequestFuture(repo *sdk.SourceCodeRepo, page api.PageInfo) {
	if page.NextPage != "" {
		g.pullrequestsFutures = append(g.pullrequestsFutures, PullRequestFuture{repo, page})
	}
}

func (g *GitlabIntegration) exportRemainingRepoPullRequests(repo *sdk.SourceCodeRepo) error {

	sdk.LogDebug(g.logger, "remaining pull requests", "repo", repo.Name)

	prs, err := g.fetchRemainingRepoPullRequests(repo)
	if err != nil {
		return err
	}

	return g.exportPullRequestEntitiesAndWrite(repo, prs)
}

func (g *GitlabIntegration) exportPullRequestEntitiesAndWrite(repo *sdk.SourceCodeRepo, prs []*api.PullRequest) (err error) {
	for _, pr := range prs {
		err = g.exportPullRequestsComments(repo, pr)
		if err != nil {
			return err
		}

		err = g.exportPullRequestsReviews(repo, pr)
		if err != nil {
			return err
		}

		err = g.exportPullRequestCommits(repo, pr)
		if err != nil {
			return err
		}
	}

	return g.writePullRequets(prs)
}

func (g *GitlabIntegration) fetchInitialRepoPullRequests(repo *sdk.SourceCodeRepo) (pi api.PageInfo, res []*api.PullRequest, rerr error) {
	params := url.Values{}
	params.Set("per_page", MaxFetchedEntitiesCount)

	return api.PullRequestPage(g.qc, repo.RefID, params)
}

func (g *GitlabIntegration) fetchRemainingRepoPullRequests(repo *sdk.SourceCodeRepo) (prs []*api.PullRequest, rerr error) {
	rerr = api.PaginateNewerThan(g.logger, "2", time.Time{}, func(log sdk.Logger, params url.Values, _ time.Time) (pi api.PageInfo, rerr error) {
		if g.lastExportDateGitlabFormat != "" {
			params.Set("updated_at", g.lastExportDateGitlabFormat)
		}
		params.Set("per_page", MaxFetchedEntitiesCount)
		pi, prs, rerr := api.PullRequestPage(g.qc, repo.RefID, params)
		if rerr != nil {
			return
		}
		for _, pr := range prs {
			prs = append(prs, pr)
		}
		return
	})
	return
}

func setPullRequestCommits(pr *sdk.SourceCodePullRequest, commits []*sdk.SourceCodePullRequestCommit) {
	commitids := []string{}
	commitshas := []string{}
	// commits come from Gitlab in the latest to earliest
	for i := len(commits) - 1; i >= 0; i-- {
		commit := commits[i]
		commitshas = append(commitshas, commit.Sha)
		commitids = append(commitids, sdk.NewSourceCodeCommitID(pr.CustomerID, commit.Sha, GitlabRefType, pr.RepoID))
	}
	pr.CommitShas = commitshas
	pr.CommitIds = commitids
	if len(commitids) > 0 {
		pr.BranchID = sdk.NewSourceCodeBranchID(GitlabRefType, pr.RepoID, pr.CustomerID, pr.BranchName, pr.CommitIds[0])
	} else {
		pr.BranchID = sdk.NewSourceCodeBranchID(GitlabRefType, pr.RepoID, pr.CustomerID, pr.BranchName, "")
	}
	for _, commit := range commits {
		commit.BranchID = pr.BranchID
	}
}

func (g *GitlabIntegration) writePullRequets(prs []*api.PullRequest) (rerr error) {
	for _, pr := range prs {
		if err := g.pipe.Write(pr.SourceCodePullRequest); err != nil {
			return err
		}
	}
	return
}

func (g *GitlabIntegration) writePullRequestCommits(commits []*sdk.SourceCodePullRequestCommit) (rerr error) {
	for _, c := range commits {
		if err := g.pipe.Write(c); err != nil {
			rerr = err
			return
		}
	}
	return
}
