package api

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

type Milestone struct {
	RefID       int64     `json:"id"`
	ProjectID   int       `json:"project_id"`
	Iid         int       `json:"iid"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	State       string    `json:"state"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DueDate     string    `json:"due_date"`
	StartDate   string    `json:"start_date"`
	WebURL      string    `json:"web_url"`
	GroupID     int       `json:"group_id"`
}

func RepoMilestonesPage(
	qc QueryContext,
	project *sdk.SourceCodeRepo,
	stopOnUpdatedAt time.Time,
	params url.Values) (pi NextPage, err error) {

	sdk.LogDebug(qc.Logger, "project work sprints", "project", project.Name, "project_ref_id", project.RefID, "params", params)

	objectPath := sdk.JoinURL("projects", url.QueryEscape(project.RefID), "milestones")

	return CommonMilestonesPage2(qc, params, stopOnUpdatedAt, objectPath, []*sdk.SourceCodeRepo{project})
}

func GroupMilestonesPage(
	qc QueryContext,
	namespace *Namespace,
	projects []*sdk.SourceCodeRepo,
	stopOnUpdatedAt time.Time,
	params url.Values) (pi NextPage, err error) {

	sdk.LogDebug(qc.Logger, "group work sprints", "group", namespace.Name, "group_ref_id", namespace.ID, "params", params)

	objectPath := sdk.JoinURL("groups", url.QueryEscape(namespace.ID), "milestones")

	return CommonMilestonesPage2(qc, params, stopOnUpdatedAt, objectPath, projects)
}

func CommonMilestonesPage2(
	qc QueryContext,
	params url.Values,
	stopOnUpdatedAt time.Time,
	url string,
	repos []*sdk.SourceCodeRepo) (pi NextPage, err error) {

	projectIDs := make([]string, 0)
	for _, repo := range repos {
		projectID := sdk.NewWorkProjectID(qc.CustomerID, repo.RefID, qc.RefType)
		projectIDs = append(projectIDs, projectID)
	}

	var rawmilestones []Milestone
	pi, err = qc.Get(url, params, &rawmilestones)
	if err != nil {
		return
	}
	for _, rawmilestone := range rawmilestones {
		if rawmilestone.UpdatedAt.Before(stopOnUpdatedAt) {
			return
		}

		// qc.WorkManager.AddMilestoneDetails(rawmilestone.RefID, rawmilestone)

		issueRefID := strconv.FormatInt(rawmilestone.RefID, 10)

		issueID := sdk.NewWorkIssueID(qc.CustomerID, issueRefID, qc.RefType)

		issue := &sdk.WorkIssue{}
		issue.ID = issueID
		issue.IntegrationInstanceID = sdk.StringPointer(qc.IntegrationInstanceID)
		issue.Active = true
		issue.CustomerID = qc.CustomerID
		issue.RefType = qc.RefType
		issue.RefID = issueRefID
		issue.Description = rawmilestone.Description

		issue.Identifier = rawmilestone.WebURL
		issue.ProjectIds = projectIDs
		issue.Title = rawmilestone.Title
		issue.Status = StatesMap[rawmilestone.State]
		issue.StatusID = sdk.NewWorkIssueStatusID(qc.CustomerID, qc.RefType, issue.Status)

		issue.Type = MilestoneIssueType
		issue.TypeID = sdk.NewWorkIssueTypeID(qc.CustomerID, qc.RefType, MilestoneIssueType)
		issue.URL = rawmilestone.WebURL

		sdk.ConvertTimeToDateModel(rawmilestone.CreatedAt, &issue.CreatedDate)
		sdk.ConvertTimeToDateModel(rawmilestone.UpdatedAt, &issue.UpdatedDate)

		if rawmilestone.DueDate != "" {
			var dueDate time.Time
			dueDate, err = time.Parse("2006-01-02", rawmilestone.DueDate)
			if err != nil {
				return
			}
			sdk.ConvertTimeToDateModel(dueDate, &issue.DueDate)
		}

		issue.Transitions = make([]sdk.WorkIssueTransitions, 0)
		if issue.Status == strings.ToLower(ClosedState) {
			issue.Transitions = append(issue.Transitions, sdk.WorkIssueTransitions{
				RefID: OpenedState,
				Name:  OpenedState,
			})
		} else {
			issue.Transitions = append(issue.Transitions, sdk.WorkIssueTransitions{
				RefID: ClosedState,
				Name:  ClosedState,
			})
		}

		err = qc.Pipe.Write(issue)
		if err != nil {
			return
		}
	}

	return
}
