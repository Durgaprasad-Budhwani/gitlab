package api

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

// Epic model
type Epic struct {
	ID           int64  `json:"id"`
	Iid          int    `json:"iid"`
	GroupID      int    `json:"group_id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Confidential bool   `json:"confidential"`
	Author       struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
		WebURL    string `json:"web_url"`
	} `json:"author"`
	// StartDate        string `json:"start_date"`
	StartDateIsFixed bool `json:"start_date_is_fixed"`
	// StartDateFixed               time.Time `json:"start_date_fixed"`
	StartDateFromInheritedSource string `json:"start_date_from_inherited_source"`
	// StartDateFromMilestones      interface{} `json:"start_date_from_milestones"`
	// EndDate        time.Time `json:"end_date"`
	DueDate        string `json:"due_date"`
	DueDateIsFixed bool   `json:"due_date_is_fixed"`
	// DueDateFixed               interface{} `json:"due_date_fixed"`
	DueDateFromInheritedSource string `json:"due_date_from_inherited_source"`
	// DueDateFromMilestones        interface{} `json:"due_date_from_milestones"`
	State      string `json:"state"`
	WebURL     string `json:"web_url"`
	References struct {
		Full string `json:"full"`
	} `json:"references"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ClosedAt  time.Time `json:"closed_at"`
	Labels    []Board   `json:"labels"`
}

// EpicsPage epics page
func EpicsPage(
	qc QueryContext,
	namespace *Namespace,
	params url.Values,
	repos []*sdk.SourceCodeRepo) (np NextPage, epics []*sdk.WorkIssue, err error) {

	sdk.LogDebug(qc.Logger, "epics page", "group_name", namespace.Name, "group_id", namespace.ID, "params", params)

	objectPath := sdk.JoinURL("groups", namespace.ID, "epics")

	var repics []Epic

	np, err = qc.Get(objectPath, params, &epics)
	if err != nil {
		return
	}

	projectIDs := make([]string, 0)
	for _, repo := range repos {
		projectID := sdk.NewWorkProjectID(qc.CustomerID, repo.RefID, "gitlab")
		projectIDs = append(projectIDs, projectID)
	}

	for _, epic := range repics {

		issueRefID := strconv.FormatInt(epic.ID, 10)
		issueID := sdk.NewWorkIssueID(qc.CustomerID, issueRefID, qc.RefType)

		issue := &sdk.WorkIssue{}
		issue.ID = issueID
		issue.Active = true
		issue.CustomerID = qc.CustomerID
		issue.RefType = qc.RefType
		issue.RefID = issueRefID

		// issue.AssigneeRefID Not supported
		issue.AssigneeRefID = strconv.FormatInt(epic.Author.ID, 10)

		issue.ReporterRefID = fmt.Sprint(epic.Author.ID)
		issue.CreatorRefID = fmt.Sprint(epic.Author.ID)

		issue.Description = epic.Description
		// issue.EpicID Not Apply
		issue.Identifier = epic.References.Full
		// issue.ProjectID Not Apply, epics are not attached to repos/projects in gitalb
		issue.ProjectID = projectIDs[0]
		issue.Title = epic.Title
		issue.Status = epic.State
		if issue.Status == "opened" {
			issue.StatusID = sdk.NewWorkIssueStatusID(qc.CustomerID, "gitlab", "1")
		} else {
			issue.StatusID = sdk.NewWorkIssueStatusID(qc.CustomerID, "gitlab", "2")
		}

		tags := make([]string, 0)
		for _, label := range epic.Labels {
			tags = append(tags, label.Name)
		}

		issue.Tags = tags
		issue.Type = "Epic"
		issue.URL = epic.WebURL

		sdk.ConvertTimeToDateModel(epic.CreatedAt, &issue.CreatedDate)
		sdk.ConvertTimeToDateModel(epic.UpdatedAt, &issue.UpdatedDate)

		// issue.SprintIds Not Apply

		if epic.StartDateFromInheritedSource != "" {
			startDate, err := time.Parse("2006-01-02", epic.StartDateFromInheritedSource)
			if err != nil {
				return np, epics, err
			}
			sdk.ConvertTimeToDateModel(startDate, &issue.PlannedStartDate)
			// sdk.ConvertTimeToDateModel(startDate, &issue.DueDate)
		}

		if epic.DueDateFromInheritedSource != "" {
			endDate, err := time.Parse("2006-01-02", epic.DueDateFromInheritedSource)
			if err != nil {
				return np, epics, err
			}
			sdk.ConvertTimeToDateModel(endDate, &issue.PlannedEndDate)
		}

		// if epic.DueDate != "" {
		// 	dueDate, err := time.Parse("2006-01-02", epic.DueDate)
		// 	if err != nil {
		// 		return np, err
		// 	}
		// 	sdk.ConvertTimeToDateModel(dueDate, &issue.DueDate)
		// }

		issue.IntegrationInstanceID = sdk.StringPointer(qc.IntegrationInstanceID)

		// pi, arr, comments, err := WorkIssuesDiscussionPage(qc, repos[0], issue.RefID, projectUsers, params)
		// if err != nil {
		// 	return pi, err
		// }
		// for _, cl := range arr {
		// 	changelogs = append(changelogs, *cl)
		// }
		// for _, c := range comments {
		// 	c.IntegrationInstanceID = ge.integrationInstanceID
		// 	if err := ge.pipe.Write(c); err != nil {
		// 		return
		// 	}
		// }

		// issue.ChangeLog = arr

		// sdk.LogDebug(qc.Logger, "writting epic", "epic", epic, "issue", issue)
		// if err := qc.Pipe.Write(&issue); err != nil {
		// 	return np, epics, err
		// }
		epics = append(epics, issue)
	}

	return np, epics, nil
}
