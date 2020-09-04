package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (ge *GitlabExport) exportIssueDiscussions(project *sdk.SourceCodeRepo, issue sdk.WorkIssue, projectUsers api.UsernameMap) (rerr error) {

	sdk.LogDebug(ge.logger, "exporting issue changelog", "issue", issue.Identifier)

	changelogs, err := ge.fetchIssueDiscussions(project, issue, projectUsers)
	if err != nil {
		return err
	}

	issue.ChangeLog = changelogs

	return
}

func (ge *GitlabExport) fetchIssueDiscussions(project *sdk.SourceCodeRepo, issue sdk.WorkIssue, projectUsers api.UsernameMap) (changelogs []sdk.WorkIssueChangeLog, rerr error) {

	rerr = api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, rerr error) {
		pi, arr, comments, err := api.WorkIssuesDiscussionPage(ge.qc, project, issue.RefID, projectUsers, params)
		if err != nil {
			return pi, err
		}
		for _, cl := range arr {
			changelogs = append(changelogs, *cl)
		}
		for _, c := range comments {
			c.IntegrationInstanceID = ge.integrationInstanceID
			if err := ge.pipe.Write(c); err != nil {
				return
			}
		}
		return
	})

	return
}

func (ge *GitlabExport) writeProjectIssues(commits []*sdk.SourceCodePullRequestCommit) (rerr error) {
	for _, c := range commits {
		c.IntegrationInstanceID = ge.integrationInstanceID
		if err := ge.pipe.Write(c); err != nil {
			rerr = err
			return
		}
	}
	return
}
