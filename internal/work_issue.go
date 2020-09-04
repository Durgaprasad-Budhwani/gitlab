package internal

import (
	"net/url"
	"sync"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (ge *GitlabExport) exportProjectIssues(project *sdk.SourceCodeRepo, users api.UsernameMap) {

	sdk.LogDebug(ge.logger, "issues", "project", project.Name)

	issuesC := make(chan sdk.WorkIssue, 10)

	done := make(chan bool, 1)
	go func() {
		ge.exportIssueEntitiesAndWrite(project, issuesC, users)
		done <- true
	}()

	go func() {
		defer close(issuesC)
		var err error
		_, err = ge.fetchInitialProjectIssues(project, issuesC)
		if err != nil {
			sdk.LogError(ge.logger, "error initial issues", "project", project.Name, "err", err)
			done <- true
		}
	}()

	<-done
}

func (ge *GitlabExport) fetchInitialProjectIssues(project *sdk.SourceCodeRepo, issues chan sdk.WorkIssue) (pi api.NextPage, rerr error) {
	params := url.Values{}

	if ge.lastExportDateGitlabFormat != "" {
		params.Set("updated_after", ge.lastExportDateGitlabFormat)
	}

	return api.WorkIssuesPage(ge.qc, project, params, issues)
}

func (ge *GitlabExport) exportIssueEntitiesAndWrite(project *sdk.SourceCodeRepo, issues chan sdk.WorkIssue, users api.UsernameMap) {

	var wg sync.WaitGroup

	for issue := range issues {
		wg.Add(1)
		go func(issue sdk.WorkIssue) {
			defer wg.Done()
			err := ge.exportIssueDiscussions(project, issue, users)
			if err != nil {
				sdk.LogError(ge.logger, "error on issue changelog", "err", err)
			}
			issue.IntegrationInstanceID = ge.integrationInstanceID
			if err = ge.pipe.Write(&issue); err != nil {
				sdk.LogError(ge.logger, "error writting issue", "err", err)
			}
		}(issue)
	}

	wg.Wait()

	return
}

func (ge *GitlabExport) exportRemainingProjectIssues(project *sdk.SourceCodeRepo, users api.UsernameMap) {

	sdk.LogDebug(ge.logger, "remaining issues", "project", project.Name)

	issuesC := make(chan sdk.WorkIssue, 10)

	done := make(chan bool, 1)
	go func() {
		ge.exportIssueEntitiesAndWrite(project, issuesC, users)
		done <- true
	}()

	go func() {
		defer close(issuesC)
		var err error
		err = ge.fetchRemainingProjectIssues(project, issuesC)
		if err != nil {
			sdk.LogError(ge.logger, "error remaining  issues", "project", project.Name, "err", err)
			done <- true
		}
	}()

	<-done
}

func (ge *GitlabExport) fetchRemainingProjectIssues(project *sdk.SourceCodeRepo, pissues chan sdk.WorkIssue) (rerr error) {
	return api.Paginate(ge.logger, "2", time.Time{}, func(log sdk.Logger, params url.Values, _ time.Time) (pi api.NextPage, rerr error) {
		if ge.lastExportDateGitlabFormat != "" {
			params.Set("updated_after", ge.lastExportDateGitlabFormat)
		}
		pi, rerr = api.WorkIssuesPage(ge.qc, project, params, pissues)
		return
	})
}
