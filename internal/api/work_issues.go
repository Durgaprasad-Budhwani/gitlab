package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

type UserModel struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	ID        int64  `json:"id"`
	AvatarURL string `json:"avatar_url"`
	WebURL    string `json:"web_url"`
}

func (u *UserModel) ToSourceCodeUser(customerID string) *sdk.SourceCodeUser {

	var userType sdk.SourceCodeUserType
	if strings.Contains(u.Name, "Bot") {
		userType = sdk.SourceCodeUserTypeBot
	} else {
		userType = sdk.SourceCodeUserTypeHuman
	}

	refID := strconv.FormatInt(u.ID, 10)

	user := &sdk.SourceCodeUser{
		Email:      sdk.StringPointer(u.Email),
		Username:   sdk.StringPointer(u.Username),
		Name:       u.Name,
		RefID:      refID,
		AvatarURL:  sdk.StringPointer(u.AvatarURL),
		URL:        sdk.StringPointer(u.WebURL),
		Type:       userType,
		CustomerID: customerID,
		RefType:    "gitlab",
	}

	return user
}

const (
	// OpenColumn open column
	OpenColumn int64 = iota
	// ClosedColumn closed column
	ClosedColumn
)

// OpenedState opened state
const OpenedState = "Opened"

// ClosedState closed state
const ClosedState = "Closed"

// StatesMap states map
var StatesMap = map[string]string{
	"opened": OpenedState,
	"closed": ClosedState,
}

// BugIssueType bug issue type
const BugIssueType = "Bug"

// EpicIssueType epic issue type
const EpicIssueType = "Epic"

// IncidentIssueType incident issue type
const IncidentIssueType = "Incident"

// EnhancementIssueType enhancement issue type
const EnhancementIssueType = "Enhancement"

func WorkIssuesPage(
	qc QueryContext,
	project *sdk.SourceCodeRepo,
	stopOnUpdatedAt time.Time,
	params url.Values,
	issues chan *sdk.WorkIssue) (pi NextPage, err error) {

	params.Set("scope", "all")
	params.Set("with_labels_details", "true")

	sdk.LogDebug(qc.Logger, "work issues", "project", project.Name, "project_ref_id", project.RefID, "params", params)

	objectPath := sdk.JoinURL("projects", url.QueryEscape(project.RefID), "issues")

	var rawissues []IssueModel

	pi, err = qc.Get(objectPath, params, &rawissues)
	if err != nil {
		return
	}

	sdk.LogDebug(qc.Logger, "issues found", "len", len(rawissues))

	for _, rawissue := range rawissues {
		if rawissue.UpdatedAt.Before(stopOnUpdatedAt) {
			return
		}

		issues <- rawissue.ToModel(qc, project.RefID)
	}

	return
}

func getIssueTypeFromLabels(tags []string, qc QueryContext) (string, string) {
	if len(tags) == 0 {
		return BugIssueType, sdk.NewWorkIssueTypeID(qc.CustomerID, qc.RefType, BugIssueType)
	}

	for _, lbl := range tags {
		switch lbl {
		case strings.ToLower(IncidentIssueType):
			// TODO: Add graphql query to get issue type as we can have issues with this label and not be a legit Incident
			return IncidentIssueType, sdk.NewWorkIssueTypeID(qc.CustomerID, qc.RefType, IncidentIssueType)
		case strings.ToLower(EnhancementIssueType):
			return EnhancementIssueType, sdk.NewWorkIssueTypeID(qc.CustomerID, qc.RefType, EnhancementIssueType)
		}
	}

	return BugIssueType, sdk.NewWorkIssueTypeID(qc.CustomerID, qc.RefType, BugIssueType)
}

type IssueWebHook struct {
	IID int64 `json:"iid"`
}

type GitlabEpic struct {
	RefID int64 `json:"id"`
}

type IssueModel struct {
	ID                 int64         `json:"id"`
	Iid                int           `json:"iid"`
	Title              string        `json:"title"`
	Description        string        `json:"description"`
	State              string        `json:"state"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
	Labels             []interface{} `json:"labels"`
	Milestone          *Milestone
	Author             UserModel   `json:"author"`
	Assignee           *UserModel  `json:"assignee"`
	UserNotesCount     int         `json:"user_notes_count"`
	MergeRequestsCount int         `json:"merge_requests_count"`
	Upvotes            int         `json:"upvotes"`
	Downvotes          int         `json:"downvotes"`
	DueDate            string      `json:"due_date"`
	Confidential       bool        `json:"confidential"`
	DiscussionLocked   interface{} `json:"discussion_locked"`
	WebURL             string      `json:"web_url"`
	TimeStats          struct {
		TimeEstimate        int         `json:"time_estimate"`
		TotalTimeSpent      int         `json:"total_time_spent"`
		HumanTimeEstimate   interface{} `json:"human_time_estimate"`
		HumanTotalTimeSpent interface{} `json:"human_total_time_spent"`
	} `json:"time_stats"`
	TaskCompletionStatus struct {
		Count          int `json:"count"`
		CompletedCount int `json:"completed_count"`
	} `json:"task_completion_status"`
	HasTasks bool `json:"has_tasks"`
	Links    struct {
		Self       string `json:"self"`
		Notes      string `json:"notes"`
		AwardEmoji string `json:"award_emoji"`
		Project    string `json:"project"`
	} `json:"_links"`
	References struct {
		Short    string `json:"short"`
		Relative string `json:"relative"`
		Full     string `json:"full"`
	} `json:"references"`
	MovedToID    interface{} `json:"moved_to_id"`
	Epic         *GitlabEpic `json:"epic"`
	Weight       *int        `json:"weight"`
	ProjectRefID int64       `json:"project_id"`
}

type IssueCreateModel struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Labels      []string `json:"labels"`
}

func (i *IssueCreateModel) ToReader() (io.Reader, error) {

	bts, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(bts), nil
}

func (i *IssueModel) ToModel(qc QueryContext, projectRefID string) *sdk.WorkIssue {

	issueRefID := strconv.FormatInt(i.ID, 10)
	issueID := sdk.NewWorkIssueID(qc.CustomerID, issueRefID, qc.RefType)

	projectID := sdk.NewWorkProjectID(qc.CustomerID, projectRefID, qc.RefType)

	item := &sdk.WorkIssue{}
	item.ID = issueID
	item.Active = true
	item.CustomerID = qc.CustomerID
	item.RefType = qc.RefType
	item.RefID = issueRefID

	if i.Assignee != nil {
		item.AssigneeRefID = fmt.Sprint(i.Assignee.ID)
	}

	item.ReporterRefID = fmt.Sprint(i.Author.ID)
	item.CreatorRefID = fmt.Sprint(i.Author.ID)
	item.Description = i.Description
	if i.Epic != nil {
		epicID := sdk.NewWorkIssueID(qc.CustomerID, strconv.FormatInt(i.Epic.RefID, 10), qc.RefType)
		item.EpicID = sdk.StringPointer(epicID)
		item.ParentID = epicID
	}
	item.Identifier = i.References.Full
	item.ProjectIds = []string{sdk.NewWorkProjectID(qc.CustomerID, projectRefID, qc.RefType)}
	item.Title = i.Title
	item.Status = StatesMap[i.State]
	item.StatusID = sdk.NewWorkIssueStatusID(qc.CustomerID, qc.RefType, item.Status)
	if i.Weight != nil {
		value := float64(*i.Weight)
		item.StoryPoints = &value
	}

	tags := make([]string, 0)

	for _, label := range i.Labels {
		switch label.(type) {
		case *Label:
			tags = append(tags, label.(*Label).Name)
		case string:
			tags = append(tags, label.(string))
		}
	}

	qc.WorkManager.AddIssue(issueID, i.State == strings.ToLower(OpenedState), projectID, i.Labels, i.Milestone, i.Assignee, i.Weight)

	item.Tags = tags
	item.Type, item.TypeID = getIssueTypeFromLabels(tags, qc)
	item.URL = i.WebURL

	sdk.ConvertTimeToDateModel(i.CreatedAt, &item.CreatedDate)
	sdk.ConvertTimeToDateModel(i.UpdatedAt, &item.UpdatedDate)

	if i.Milestone != nil {
		item.SprintIds = []string{sdk.NewAgileSprintID(qc.CustomerID, strconv.FormatInt(int64(i.Milestone.RefID), 10), qc.RefType)}

		duedate, err := time.Parse("2006-01-02", i.Milestone.DueDate)
		if err != nil {
			duedate = time.Time{}
		}
		sdk.ConvertTimeToDateModel(duedate, &item.PlannedEndDate)

		startdate, err := time.Parse("2006-01-02", i.Milestone.StartDate)
		if err != nil {
			startdate = time.Time{}
		}
		sdk.ConvertTimeToDateModel(startdate, &item.PlannedStartDate)
	}

	return item
}

// CreateWorkIssue create work issue
func CreateWorkIssue(qc QueryContext, mutation *sdk.WorkIssueCreateMutation, label string) (*sdk.MutationResponse, error) {

	sdk.LogDebug(qc.Logger, "create issue", "project_ref_id", mutation.ProjectRefID, "body", sdk.Stringify(mutation))

	objectPath := sdk.JoinURL("projects", mutation.ProjectRefID, "issues")

	issueCreate := convertMutationToGitlabIssue(mutation)
	if label != "" {
		issueCreate.Labels = []string{strings.ToLower(label)}
	}

	reader, err := issueCreate.ToReader()
	if err != nil {
		return nil, err
	}

	var issue IssueModel

	_, err = qc.Post(objectPath, nil, reader, &issue)
	if err != nil {
		return nil, err
	}

	workIssue := issue.ToModel(qc, mutation.ProjectRefID)

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

func convertMutationToGitlabIssue(m *sdk.WorkIssueCreateMutation) IssueCreateModel {
	return IssueCreateModel{
		Title:       m.Title,
		Description: m.Description,
	}
}
