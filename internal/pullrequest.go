package internal

import (
	"net/url"
	"sync"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

// PullRequestFuture pull requests will process later
type PullRequestFuture struct {
	Repo *sdk.SourceCodeRepo
}

func (g *GitlabIntegration) exportRepoPullRequests(repo *sdk.SourceCodeRepo) {

	sdk.LogDebug(g.logger, "pull requests", "repo", repo.Name)

	prsChan := make(chan api.PullRequest, 10)

	done := make(chan bool, 1)
	go func() {
		g.exportPullRequestEntitiesAndWrite(repo, prsChan)
		done <- true
	}()

	page := &api.PageInfo{}
	go func() {
		defer close(prsChan)
		var err error
		*page, err = g.fetchInitialRepoPullRequests(repo, prsChan)
		if err != nil {
			sdk.LogError(g.logger, "error initial pull requests", "repo", repo.Name, "err", err)
			done <- true
		}
	}()

	<-done
	g.addPullRequestFuture(repo, page)
}

func (g *GitlabIntegration) exportRemainingRepoPullRequests(repo *sdk.SourceCodeRepo) {

	sdk.LogDebug(g.logger, "remaining pull requests", "repo", repo.Name)

	prsChan := make(chan api.PullRequest, 10)

	done := make(chan bool, 1)
	go func() {
		g.exportPullRequestEntitiesAndWrite(repo, prsChan)
		done <- true
	}()

	go func() {
		defer close(prsChan)
		var err error
		err = g.fetchRemainingRepoPullRequests(repo, prsChan)
		if err != nil {
			sdk.LogError(g.logger, "error remaining  pull requests", "repo", repo.Name, "err", err)
			done <- true
		}
	}()

	<-done
}

func (g *GitlabIntegration) exportPullRequestEntitiesAndWrite(repo *sdk.SourceCodeRepo, prs chan api.PullRequest) {

	var wg sync.WaitGroup

	for pr := range prs {
		wg.Add(1)
		go func(pr api.PullRequest) {
			defer wg.Done()

			err := g.exportPullRequestsComments(repo, pr)
			if err != nil {
				sdk.LogError(g.logger, "error on pull request comments", "err", err)
			}

			err = g.exportPullRequestsReviews(repo, pr)
			if err != nil {
				sdk.LogError(g.logger, "error on pull request reviews", "err", err)
			}

			err = g.exportPullRequestCommits(repo, pr)
			if err != nil {
				sdk.LogError(g.logger, "error on pull request commits", "err", err)
			}

			sdk.LogDebug(g.logger, "pull request done", "identifier", pr.Identifier, "title", pr.Title)
			if err := g.pipe.Write(pr.SourceCodePullRequest); err != nil {
				sdk.LogError(g.logger, "error writting pr", "err", err)
			}
		}(pr)
	}

	wg.Wait()

}

func (g *GitlabIntegration) addPullRequestFuture(repo *sdk.SourceCodeRepo, page *api.PageInfo) {
	if page != nil && page.NextPage != "" {
		g.pullrequestsFutures = append(g.pullrequestsFutures, PullRequestFuture{repo})
	}
}

func (g *GitlabIntegration) fetchInitialRepoPullRequests(repo *sdk.SourceCodeRepo, prs chan api.PullRequest) (pi api.PageInfo, rerr error) {
	params := url.Values{}
	params.Set("per_page", MaxFetchedEntitiesCount)

	if g.lastExportDateGitlabFormat != "" {
		params.Set("updated_after", g.lastExportDateGitlabFormat)
	}

	return api.PullRequestPage(g.qc, repo, params, prs)
}

func (g *GitlabIntegration) fetchRemainingRepoPullRequests(repo *sdk.SourceCodeRepo, prs chan api.PullRequest) (rerr error) {
	rerr = api.PaginateNewerThan(g.logger, "2", time.Time{}, func(log sdk.Logger, params url.Values, _ time.Time) (pi api.PageInfo, rerr error) {
		if g.lastExportDateGitlabFormat != "" {
			params.Set("updated_after", g.lastExportDateGitlabFormat)
		}
		params.Set("per_page", MaxFetchedEntitiesCount)
		pi, rerr = api.PullRequestPage(g.qc, repo, params, prs)
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

func (g *GitlabIntegration) writePullRequestCommits(commits []*sdk.SourceCodePullRequestCommit) (rerr error) {
	for _, c := range commits {
		if err := g.pipe.Write(c); err != nil {
			rerr = err
			return
		}
	}
	return
}
