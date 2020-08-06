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

func (ge *GitlabExport) exportRepoPullRequests(repo *sdk.SourceCodeRepo) {

	sdk.LogDebug(ge.logger, "pull requests", "repo", repo.Name)

	prsChan := make(chan api.PullRequest, 10)

	done := make(chan bool, 1)
	go func() {
		ge.exportPullRequestEntitiesAndWrite(repo, prsChan)
		done <- true
	}()

	var np api.NextPage
	go func() {
		defer close(prsChan)
		var err error
		np, err = ge.fetchInitialRepoPullRequests(repo, prsChan)
		if err != nil {
			sdk.LogError(ge.logger, "error initial pull requests", "repo", repo.Name, "err", err)
			done <- true
		}
	}()

	<-done
	ge.addPullRequestFuture(repo, np)
}

func (ge *GitlabExport) exportRemainingRepoPullRequests(repo *sdk.SourceCodeRepo) {

	sdk.LogDebug(ge.logger, "remaining pull requests", "repo", repo.Name)

	prsChan := make(chan api.PullRequest, 10)

	done := make(chan bool, 1)
	go func() {
		ge.exportPullRequestEntitiesAndWrite(repo, prsChan)
		done <- true
	}()

	go func() {
		defer close(prsChan)
		var err error
		err = ge.fetchRemainingRepoPullRequests(repo, prsChan)
		if err != nil {
			sdk.LogError(ge.logger, "error remaining  pull requests", "repo", repo.Name, "err", err)
			done <- true
		}
	}()

	<-done
}

func (ge *GitlabExport) exportPullRequestEntitiesAndWrite(repo *sdk.SourceCodeRepo, prs chan api.PullRequest) {

	var wg sync.WaitGroup

	for pr := range prs {
		wg.Add(1)
		go func(pr api.PullRequest) {
			defer wg.Done()

			err := ge.exportPullRequestsComments(repo, pr)
			if err != nil {
				sdk.LogError(ge.logger, "error on pull request comments", "err", err)
			}

			err = ge.exportPullRequestsReviews(repo, pr)
			if err != nil {
				sdk.LogError(ge.logger, "error on pull request reviews", "err", err)
			}

			err = ge.exportPullRequestCommits(repo, pr)
			if err != nil {
				sdk.LogError(ge.logger, "error on pull request commits", "err", err)
			}

			sdk.LogDebug(ge.logger, "pull request done", "identifier", pr.Identifier, "title", pr.Title)
			pr.IntegrationInstanceID = ge.integrationInstanceID
			if err := ge.pipe.Write(pr.SourceCodePullRequest); err != nil {
				sdk.LogError(ge.logger, "error writting pr", "err", err)
			}
		}(pr)
	}

	wg.Wait()

}

func (ge *GitlabExport) addPullRequestFuture(repo *sdk.SourceCodeRepo, np api.NextPage) {
	if string(np) != "" {
		ge.pullrequestsFutures = append(ge.pullrequestsFutures, PullRequestFuture{repo})
	}
}

func (ge *GitlabExport) fetchInitialRepoPullRequests(repo *sdk.SourceCodeRepo, prs chan api.PullRequest) (pi api.NextPage, rerr error) {
	params := url.Values{}

	if ge.lastExportDateGitlabFormat != "" {
		params.Set("updated_after", ge.lastExportDateGitlabFormat)
	}

	return api.PullRequestPage(ge.qc, repo, params, prs)
}

func (ge *GitlabExport) fetchRemainingRepoPullRequests(repo *sdk.SourceCodeRepo, prs chan api.PullRequest) (rerr error) {
	rerr = api.Paginate(ge.logger, "2", time.Time{}, func(log sdk.Logger, params url.Values, _ time.Time) (pi api.NextPage, rerr error) {
		if ge.lastExportDateGitlabFormat != "" {
			params.Set("updated_after", ge.lastExportDateGitlabFormat)
		}
		pi, rerr = api.PullRequestPage(ge.qc, repo, params, prs)
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
		commitids = append(commitids, sdk.NewSourceCodeCommitID(pr.CustomerID, commit.Sha, gitlabRefType, pr.RepoID))
	}
	pr.CommitShas = commitshas
	pr.CommitIds = commitids
	if len(commitids) > 0 {
		pr.BranchID = sdk.NewSourceCodeBranchID(gitlabRefType, pr.RepoID, pr.CustomerID, pr.BranchName, pr.CommitIds[0])
	} else {
		pr.BranchID = sdk.NewSourceCodeBranchID(gitlabRefType, pr.RepoID, pr.CustomerID, pr.BranchName, "")
	}
	for _, commit := range commits {
		commit.BranchID = pr.BranchID
	}
}

func (ge *GitlabExport) writePullRequestCommits(commits []*sdk.SourceCodePullRequestCommit) (rerr error) {
	for _, c := range commits {
		c.IntegrationInstanceID = ge.integrationInstanceID
		if err := ge.pipe.Write(c); err != nil {
			rerr = err
			return
		}
	}
	return
}
