package internal

import (
	"net/url"
	"sync"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

// IssueFuture issues will process later
type IssueFuture struct {
	Project *sdk.WorkProject
}

func (g *GitlabIntegration) exportProjectIssues(project *sdk.WorkProject, users api.UsernameMap) {

	sdk.LogDebug(g.logger, "issues", "project", project.Name)

	issuesC := make(chan sdk.WorkIssue, 10)

	done := make(chan bool, 1)
	go func() {
		g.exportIssueEntitiesAndWrite(project, issuesC, users)
		done <- true
	}()

	var np api.NextPage
	go func() {
		defer close(issuesC)
		var err error
		np, err = g.fetchInitialProjectIssues(project, issuesC)
		if err != nil {
			sdk.LogError(g.logger, "error initial issues", "project", project.Name, "err", err)
			done <- true
		}
	}()

	<-done
	g.addIssueFuture(project, np)
}

func (g *GitlabIntegration) fetchInitialProjectIssues(project *sdk.WorkProject, issues chan sdk.WorkIssue) (pi api.NextPage, rerr error) {
	params := url.Values{}

	if g.lastExportDateGitlabFormat != "" {
		params.Set("updated_after", g.lastExportDateGitlabFormat)
	}

	return api.WorkIssuesPage(g.qc, project, params, issues)
}

func (g *GitlabIntegration) addIssueFuture(project *sdk.WorkProject, np api.NextPage) {
	if string(np) != "" {
		g.isssueFutures = append(g.isssueFutures, IssueFuture{project})
	}
}

func (g *GitlabIntegration) exportIssueEntitiesAndWrite(project *sdk.WorkProject, issues chan sdk.WorkIssue, users api.UsernameMap) {

	var wg sync.WaitGroup

	for issue := range issues {
		wg.Add(1)
		go func(issue sdk.WorkIssue) {
			defer wg.Done()
			err := g.exportIssueDiscussions(project, issue, users)
			if err != nil {
				sdk.LogError(g.logger, "error on issue changelog", "err", err)
			}
			if err = g.pipe.Write(&issue); err != nil {
				sdk.LogError(g.logger, "error writting pr", "err", err)
			}
		}(issue)
	}

	wg.Wait()

	return
}

func (g *GitlabIntegration) exportRemainingProjectIssues(project *sdk.WorkProject, users api.UsernameMap) {

	sdk.LogDebug(g.logger, "remaining issues", "project", project.Name)

	issuesC := make(chan sdk.WorkIssue, 10)

	done := make(chan bool, 1)
	go func() {
		g.exportIssueEntitiesAndWrite(project, issuesC, users)
		done <- true
	}()

	go func() {
		defer close(issuesC)
		var err error
		err = g.fetchRemainingProjectIssues(project, issuesC)
		if err != nil {
			sdk.LogError(g.logger, "error remaining  issues", "project", project.Name, "err", err)
			done <- true
		}
	}()

	<-done
}

func (g *GitlabIntegration) fetchRemainingProjectIssues(project *sdk.WorkProject, pissues chan sdk.WorkIssue) (rerr error) {
	return api.Paginate(g.logger, "2", time.Time{}, func(log sdk.Logger, params url.Values, _ time.Time) (pi api.NextPage, rerr error) {
		if g.lastExportDateGitlabFormat != "" {
			params.Set("updated_after", g.lastExportDateGitlabFormat)
		}
		pi, rerr = api.WorkIssuesPage(g.qc, project, params, pissues)
		return
	})
}
