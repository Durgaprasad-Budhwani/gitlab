package internal

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

func (ge *GitlabExport) exportIssueDiscussions(project *sdk.SourceCodeRepo, issue *sdk.WorkIssue, projectUsers api.UsernameMap) (rerr error) {

	sdk.LogDebug(ge.logger, "exporting issue changelog", "issue", issue.Identifier)

	changelogs, err := ge.fetchIssueDiscussions(project, issue, projectUsers)
	if err != nil {
		return err
	}

	index := strings.Index(issue.Identifier, "#")
	if index == -1 {
		sdk.LogWarn(ge.logger, "no issue iid found", project.Name, "project_ref_id", project.RefID, "issue", issue)
		return
	}
	issueIID := issue.Identifier[index+1:]

	sdk.LogDebug(ge.logger, "work issues changelog resource_state_events", "project", project.RefID)

	stateEvents, err := api.GetOpenClosedIssueHistory(ge.qc, project, issueIID)
	if err != nil {
		return err
	}

	transitions := make([]sdk.WorkIssueTransitions, 0)
	for _, stateEvent := range stateEvents {
		changelog := sdk.WorkIssueChangeLog{
			RefID:  fmt.Sprint(stateEvent.ID),
			UserID: strconv.FormatInt(stateEvent.User.ID, 10),
		}
		sdk.ConvertTimeToDateModel(stateEvent.CreatedAt, &changelog.CreatedDate)

		transition := sdk.WorkIssueTransitions{}
		if stateEvent.State == "closed" {
			changelog.To = stateEvent.State
			changelog.ToString = stateEvent.State
			changelog.From = "1"
			changelog.Field = sdk.WorkIssueChangeLogFieldStatus

			transition.RefID = "2"
			transition.Name = stateEvent.State
		} else if stateEvent.State == "reopened" {
			changelog.To = "opened"
			changelog.ToString = "opened"
			changelog.Field = sdk.WorkIssueChangeLogFieldStatus

			transition.RefID = "1"
			transition.Name = "opened"
		}
		changelogs = append(changelogs, changelog)
		transitions = append(transitions, transition)
	}

	issue.ChangeLog = changelogs

	if len(transitions) == 0 {
		issue.Transitions = []sdk.WorkIssueTransitions{
			{
				RefID: "1",
				Name:  "opened",
			},
		}
	} else {
		issue.Transitions = transitions
	}

	return
}

func (ge *GitlabExport) fetchIssueDiscussions(project *sdk.SourceCodeRepo, issue *sdk.WorkIssue, projectUsers api.UsernameMap) (changelogs []sdk.WorkIssueChangeLog, rerr error) {

	rerr = api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (np api.NextPage, rerr error) {
		pi, arr, comments, err := api.WorkIssuesDiscussionPage(ge.qc, project, issue, projectUsers, params)
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
