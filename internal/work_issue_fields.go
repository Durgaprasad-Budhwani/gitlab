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

func (ge *GitlabExport) exportIssueFields(project *api.GitlabProjectInternal, issue *sdk.WorkIssue, projectUsers api.UsernameMap) (rerr error) {

	sdk.LogDebug(ge.logger, "exporting issue changelog", "issue", issue.Identifier)

	changelogs, err := ge.fetchIssueDiscussions(project, issue, projectUsers)
	if err != nil {
		return fmt.Errorf("error on issue changelog, %s", err)
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

	ordinal := sdk.EpochNow()
	for _, stateEvent := range stateEvents {
		ordinal++
		changelog := sdk.WorkIssueChangeLog{
			RefID:   fmt.Sprint(stateEvent.ID),
			UserID:  strconv.FormatInt(stateEvent.User.ID, 10),
			Field:   sdk.WorkIssueChangeLogFieldStatus,
			Ordinal: ordinal,
		}
		sdk.ConvertTimeToDateModel(stateEvent.CreatedAt, &changelog.CreatedDate)

		if stateEvent.State == strings.ToLower(api.ClosedState) {
			changelog.To = api.ClosedState
			changelog.ToString = api.ClosedState

			changelog.From = api.OpenedState
			changelog.FromString = api.OpenedState
		} else if stateEvent.State == "reopened" {
			changelog.To = api.OpenedState
			changelog.ToString = api.OpenedState

			changelog.From = api.ClosedState
			changelog.FromString = api.ClosedState
		}
		changelogs = append(changelogs, changelog)
	}

	issue.ChangeLog = changelogs

	transition := sdk.WorkIssueTransitions{}
	if issue.Status == "reopened" {
		transition.RefID = api.ClosedState
		transition.Name = api.ClosedState
	} else {
		transition.RefID = api.OpenedState
		transition.Name = api.OpenedState
	}

	issue.Transitions = []sdk.WorkIssueTransitions{transition}

	links, err := api.GetIssueLinks(ge.qc, project, issueIID)
	if err != nil {
		return fmt.Errorf("error on issue links, %s", err)
	}

	issue.LinkedIssues = links

	attachments, err := api.GetIssueAttachments(ge.qc, project, issue.RefID)
	if err != nil {
		return fmt.Errorf("error on issue attachments, %s issue %s", err, issue.RefID)
	}

	issue.Attachments = attachments

	return
}

func (ge *GitlabExport) fetchIssueDiscussions(project *api.GitlabProjectInternal, issue *sdk.WorkIssue, projectUsers api.UsernameMap) (changelogs []sdk.WorkIssueChangeLog, rerr error) {

	rerr = api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (np api.NextPage, rerr error) {
		np, arr, comments, err := api.WorkIssuesDiscussionPage(ge.qc, project, issue, projectUsers, params)
		if err != nil {
			return np, err
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

func (ge *GitlabExport) fetchEpicIssueDiscussions(namespace *api.Namespace, projects []*api.GitlabProjectInternal, epic *sdk.WorkIssue, projectUsers api.UsernameMap) (changelogs []sdk.WorkIssueChangeLog, err error) {

	err = api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (api.NextPage, error) {
		pi, arr, comments, err := api.WorkEpicIssuesDiscussionPage(ge.qc, namespace, projects, epic, projectUsers, params)
		if err != nil {
			return pi, err
		}
		for _, cl := range arr {
			changelogs = append(changelogs, *cl)
		}
		for _, c := range comments {
			c.IntegrationInstanceID = ge.integrationInstanceID
			if err := ge.pipe.Write(c); err != nil {
				return pi, err
			}
		}
		return pi, nil
	})

	return
}
