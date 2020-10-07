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

		issueRefID := strconv.FormatInt(epic.RefID, 10)
		issueID := sdk.NewWorkIssueID(qc.CustomerID, issueRefID, qc.RefType)

		issue := &sdk.WorkIssue{}
		issue.ID = issueID
		issue.Active = true
		issue.CustomerID = qc.CustomerID
		issue.RefType = qc.RefType
		issue.RefID = issueRefID

		// issue.AssigneeRefID Not supported

		issue.ReporterRefID = fmt.Sprint(epic.Author.ID)
		issue.CreatorRefID = fmt.Sprint(epic.Author.ID)

		issue.Description = epic.Description
		if epic.ParentID != nil {
			epicID := sdk.NewWorkIssueID(qc.CustomerID, strconv.FormatInt(*epic.ParentID, 10), qc.RefType)
			issue.EpicID = sdk.StringPointer(epicID)
			issue.ParentID = epicID
		}
		issue.Identifier = epic.References.Full
		issue.ProjectIds = projectIDs
		issue.Title = epic.Title
		issue.Status = StatesMap[epic.State]
		issue.StatusID = sdk.NewWorkIssueStatusID(qc.CustomerID, qc.RefType, issue.Status)

		tags := make([]string, 0)
		for _, labelName := range epic.Labels {
			tags = append(tags, labelName)
		}

		issue.Tags = tags
		issue.Type = EpicIssueType
		issue.TypeID = sdk.NewWorkIssueTypeID(qc.CustomerID, qc.RefType, EpicIssueType)
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
		}

		if epic.DueDateFromInheritedSource != "" {
			endDate, err := time.Parse("2006-01-02", epic.DueDateFromInheritedSource)
			if err != nil {
				return np, epics, err
			}
			sdk.ConvertTimeToDateModel(endDate, &issue.PlannedEndDate)
		}

		issue.IntegrationInstanceID = sdk.StringPointer(qc.IntegrationInstanceID)

		epics = append(epics, issue)
	}

	return np, epics, nil
}
