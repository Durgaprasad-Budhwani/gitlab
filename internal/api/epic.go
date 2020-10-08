package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

// Epic model
type Epic struct {
	RefID        int64  `json:"id"`
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
	StartDateIsFixed             bool   `json:"start_date_is_fixed"`
	StartDateFromInheritedSource string `json:"start_date_from_inherited_source"`
	DueDate                      string `json:"due_date"`
	DueDateIsFixed               bool   `json:"due_date_is_fixed"`
	DueDateFromInheritedSource   string `json:"due_date_from_inherited_source"`
	State                        string `json:"state"`
	WebURL                       string `json:"web_url"`
	References                   struct {
		Full string `json:"full"`
	} `json:"references"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ClosedAt  time.Time `json:"closed_at"`
	Labels    []string  `json:"labels"`
	ParentID  *int64    `json:"parent_id"`
}

func (e *Epic) ToModel(qc QueryContext, projectIDs []string) (*sdk.WorkIssue, error) {

	issueRefID := strconv.FormatInt(e.RefID, 10)
	issueID := sdk.NewWorkIssueID(qc.CustomerID, issueRefID, qc.RefType)

	issue := &sdk.WorkIssue{}
	issue.ID = issueID
	issue.Active = true
	issue.CustomerID = qc.CustomerID
	issue.RefType = qc.RefType
	issue.RefID = issueRefID

	issue.ReporterRefID = fmt.Sprint(e.Author.ID)
	issue.CreatorRefID = fmt.Sprint(e.Author.ID)

	issue.Description = e.Description
	if e.ParentID != nil {
		epicID := sdk.NewWorkIssueID(qc.CustomerID, strconv.FormatInt(*e.ParentID, 10), qc.RefType)
		issue.EpicID = sdk.StringPointer(epicID)
		issue.ParentID = epicID
	}
	issue.Identifier = e.References.Full
	issue.ProjectIds = projectIDs
	issue.Title = e.Title
	issue.Status = StatesMap[e.State]
	issue.StatusID = sdk.NewWorkIssueStatusID(qc.CustomerID, qc.RefType, issue.Status)

	tags := make([]string, 0)
	for _, labelName := range e.Labels {
		tags = append(tags, labelName)
	}

	issue.Tags = tags
	issue.Type = EpicIssueType
	issue.TypeID = sdk.NewWorkIssueTypeID(qc.CustomerID, qc.RefType, EpicIssueType)
	issue.URL = e.WebURL

	sdk.ConvertTimeToDateModel(e.CreatedAt, &issue.CreatedDate)
	sdk.ConvertTimeToDateModel(e.UpdatedAt, &issue.UpdatedDate)

	// issue.SprintIds Not Apply

	if e.StartDateFromInheritedSource != "" {
		startDate, err := time.Parse("2006-01-02", e.StartDateFromInheritedSource)
		if err != nil {
			return nil, err
		}
		sdk.ConvertTimeToDateModel(startDate, &issue.PlannedStartDate)
	}

	if e.DueDateFromInheritedSource != "" {
		endDate, err := time.Parse("2006-01-02", e.DueDateFromInheritedSource)
		if err != nil {
			return nil, err
		}
		sdk.ConvertTimeToDateModel(endDate, &issue.PlannedEndDate)
	}

	issue.IntegrationInstanceID = sdk.StringPointer(qc.IntegrationInstanceID)

	return issue, nil
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

	np, err = qc.Get(objectPath, params, &repics)
	if err != nil {
		if strings.Contains(err.Error(), "403") {
			sdk.LogWarn(qc.Logger, "epics is not available for this namespace, it needs a valid tier", "namespace", namespace.Name)
			return np, epics, nil
		}
		return
	}

	projectIDs := make([]string, 0)
	for _, repo := range repos {
		projectID := sdk.NewWorkProjectID(qc.CustomerID, repo.RefID, qc.RefType)
		projectIDs = append(projectIDs, projectID)
	}

	for _, epic := range repics {
		e, err := epic.ToModel(qc, projectIDs)
		if err != nil {
			return np, epics, err
		}
		epics = append(epics, e)
	}

	return np, epics, nil
}

// CreateEpic create epic
func CreateEpic(qc QueryContext, mutation *sdk.WorkIssueCreateMutation) (*sdk.MutationResponse, error) {

	sdk.LogDebug(qc.Logger, "create epic", "project_ref_id", mutation.ProjectRefID)

	projectRefID, err := strconv.Atoi(mutation.ProjectRefID)
	if err != nil {
		return nil, err
	}

	repo, err := ProjectByRefID(qc, int64(projectRefID))
	if err != nil {
		return nil, err
	}

	ind := strings.Index(repo.Name, "/")

	groupName := repo.Name[:ind]

	objectPath := sdk.JoinURL("groups", url.QueryEscape(groupName), "epics")

	issueCreate := convertMutationToGitlabIssue(mutation)

	reader, err := issueCreate.ToReader()
	if err != nil {
		return nil, err
	}

	var epic Epic

	_, err = qc.Post(objectPath, nil, reader, &epic)
	if err != nil {
		return nil, err
	}

	projectIDs, err := GroupProjectsIDs(qc, &Namespace{
		ID:   groupName,
		Name: groupName,
	})
	if err != nil {
		return nil, err
	}

	workIssue, err := epic.ToModel(qc, projectIDs)
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
