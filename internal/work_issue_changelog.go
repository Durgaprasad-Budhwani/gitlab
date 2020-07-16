package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (g *GitlabIntegration) exportIssueDiscussions(project *sdk.WorkProject, issue sdk.WorkIssue, projectUsers api.UsernameMap) (rerr error) {

	sdk.LogDebug(g.logger, "exporting issue changelog", "issue", issue.Identifier)

	changelogs, err := g.fetchIssueDiscussions(project, issue, projectUsers)
	if err != nil {
		return err
	}

	issue.ChangeLog = changelogs

	return
}

func (g *GitlabIntegration) fetchIssueDiscussions(project *sdk.WorkProject, issue sdk.WorkIssue, projectUsers api.UsernameMap) (changelogs []sdk.WorkIssueChangeLog, rerr error) {

	rerr = api.Paginate(g.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, rerr error) {
		pi, arr, comments, err := api.WorkIssuesDiscussionPage(g.qc, project, issue.RefID, projectUsers, params)
		if err != nil {
			return pi, err
		}
		for _, cl := range arr {
			changelogs = append(changelogs, *cl)
		}
		for _, c := range comments {
			if err := g.pipe.Write(c); err != nil {
				return
			}
		}
		return
	})

	return
}

func (g *GitlabIntegration) writeProjectIssues(commits []*sdk.SourceCodePullRequestCommit) (rerr error) {
	for _, c := range commits {
		if err := g.pipe.Write(c); err != nil {
			rerr = err
			return
		}
	}
	return
}
