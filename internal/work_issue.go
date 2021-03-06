package internal

import (
	"net/url"
	"strconv"
	"sync"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

func (ge *GitlabExport) exportIssueEntitiesAndWrite(project *api.GitlabProjectInternal, issues chan *sdk.WorkIssue, users api.UsernameMap) {

	var wg sync.WaitGroup

	for issue := range issues {
		wg.Add(1)
		go func(issue *sdk.WorkIssue) {
			defer wg.Done()
			err := ge.exportIssueFields(project, issue, users)
			if err != nil {
				sdk.LogError(ge.logger, "error on issue fields", "err", err)
			}
			issue.IntegrationInstanceID = ge.integrationInstanceID
			if err := ge.pipe.Write(issue); err != nil {
				sdk.LogError(ge.logger, "error writting issue", "err", err)
			}
		}(issue)
	}

	wg.Wait()

	return
}

func (ge *GitlabExport) exportProjectIssues(project *api.GitlabProjectInternal, users api.UsernameMap) {

	sdk.LogDebug(ge.logger, "exporting project issues", "project", project.Name)

	issuesC := make(chan *sdk.WorkIssue, 10)

	done := make(chan bool, 1)
	go func() {
		ge.exportIssueEntitiesAndWrite(project, issuesC, users)
		done <- true
	}()

	go func() {
		defer close(issuesC)
		var err error
		err = ge.fetchProjectIssues(project, issuesC)
		if err != nil {
			sdk.LogError(ge.logger, "error exporting project issues", "project", project.Name, "err", err)
			done <- true
		}
	}()

	<-done
}

func (ge *GitlabExport) fetchProjectIssues(project *api.GitlabProjectInternal, pissues chan *sdk.WorkIssue) (err error) {
	var nP api.NextPage
	for {
		nP, err = api.WorkIssuesPage(ge.qc, project, nP, pissues)
		if err != nil {
			return err
		}
		if nP == "" {
			return
		}
	}
}

func (ge *GitlabExport) writeSingleIssue(project *sdk.SourceCodeRepo, iid int64) error {

	params := url.Values{}
	params.Set("iids[]", strconv.FormatInt(iid, 10))

	issuesC := make(chan *sdk.WorkIssue, 1)
	_, err := api.WorkSingleIssue(ge.qc, project, ge.lastExportDate, params, issuesC)
	if err != nil {
		return err
	}
	issue := <-issuesC

	issue.IntegrationInstanceID = ge.integrationInstanceID

	return ge.qc.Pipe.Write(issue)
}
