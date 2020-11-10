package api

import (
	"fmt"
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

func (m *Milestone) ToModel(customerID, integrationInstanceID string, projectIDs []string) (*sdk.WorkIssue, error) {

	issueRefID := strconv.FormatInt(m.RefID, 10)

	issueID := sdk.NewWorkIssueID(customerID, issueRefID, gitlabRefType)

	issue := &sdk.WorkIssue{}
	issue.ID = issueID
	issue.IntegrationInstanceID = sdk.StringPointer(integrationInstanceID)
	issue.Active = true
	issue.CustomerID = customerID
	issue.RefType = gitlabRefType
	issue.RefID = issueRefID
	issue.Description = m.Description

	issue.Identifier = fmt.Sprintf("%s#%d", m.Title, m.RefID)
	issue.ProjectIds = projectIDs
	issue.Title = m.Title
	issue.Status = StatesMap[m.State]
	issue.StatusID = sdk.NewWorkIssueStatusID(customerID, gitlabRefType, issue.Status)

	issue.Type = MilestoneIssueType
	issue.TypeID = sdk.NewWorkIssueTypeID(customerID, gitlabRefType, MilestoneIssueType)
	issue.URL = m.WebURL

	sdk.ConvertTimeToDateModel(m.CreatedAt, &issue.CreatedDate)
	sdk.ConvertTimeToDateModel(m.UpdatedAt, &issue.UpdatedDate)

	if m.DueDate != "" {
		var dueDate time.Time
		dueDate, err := time.Parse("2006-01-02", m.DueDate)
		if err != nil {
			return nil, err
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

	return issue, nil
}

func RepoMilestonesPage(
	qc QueryContext,
	project *GitlabProjectInternal,
	stopOnUpdatedAt time.Time,
	params url.Values) (pi NextPage, err error) {

	sdk.LogDebug(qc.Logger, "project work sprints", "project", project.Name, "project_ref_id", project.RefID, "params", params)

	objectPath := sdk.JoinURL("projects", url.QueryEscape(project.RefID), "milestones")

	return CommonMilestonesPage2(qc, params, stopOnUpdatedAt, objectPath, []*GitlabProjectInternal{project})
}

func GroupMilestonesPage(
	qc QueryContext,
	namespace *Namespace,
	projects []*GitlabProjectInternal,
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
	repos []*GitlabProjectInternal) (pi NextPage, err error) {

	projectIDs := make([]string, 0)
	for _, repo := range repos {
		projectID := sdk.NewWorkProjectID(qc.CustomerID, repo.RefID, qc.RefType)
		projectIDs = append(projectIDs, projectID)
	}

	var rawmilestones []Milestone
	pi, err = qc.Get(url, params, &rawmilestones)
	if err != nil {
		return pi, err
	}
	for _, rawmilestone := range rawmilestones {
		if rawmilestone.UpdatedAt.Before(stopOnUpdatedAt) {
			return pi, nil
		}

		issue, err := rawmilestone.ToModel(qc.CustomerID, qc.IntegrationInstanceID, projectIDs)
		if err != nil {
			return pi, err
		}

		err = qc.Pipe.Write(issue)
		if err != nil {
			return pi, err
		}
	}

	return pi, nil
}

// CreateMilestone create milestone
func CreateMilestone(qc QueryContext, body map[string]interface{}, projectName, projectRefID string) (*sdk.MutationResponse, error) {

	sdk.LogDebug(qc.Logger, "create milestone", "project_ref_id", projectRefID)

	objectPath := sdk.JoinURL("projects", projectRefID, "epics")

	var milestone Milestone

	_, err := qc.Post(objectPath, nil, sdk.StringifyReader(body), &milestone)
	if err != nil {
		return nil, err
	}

	projectID := sdk.NewWorkProjectID(qc.CustomerID, projectRefID, qc.RefType)

	workIssue, err := milestone.ToModel(qc.CustomerID, qc.IntegrationInstanceID, []string{projectID})
	if err != nil {
		return nil, err
	}

	transition := sdk.WorkIssueTransitions{}
	transition.RefID = ClosedState
	transition.Name = ClosedState

	workIssue.Transitions = []sdk.WorkIssueTransitions{transition}

	err = qc.Pipe.Write(workIssue)
	if err != nil {
		return nil, err
	}

	return &sdk.MutationResponse{
		RefID:    sdk.StringPointer(workIssue.RefID),
		EntityID: sdk.StringPointer(workIssue.ID),
		URL:      sdk.StringPointer(workIssue.URL),
	}, nil

}
