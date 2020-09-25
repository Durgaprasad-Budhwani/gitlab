package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pinpt/agent.next/sdk"
)

const (
	OpenColumn int64 = iota
	ClosedColumn
)

func WorkIssuesPage(
	qc QueryContext,
	project *sdk.SourceCodeRepo,
	stopOnUpdatedAt time.Time,
	params url.Values,
	issues chan sdk.WorkIssue) (pi NextPage, err error) {

	params.Set("scope", "all")
	params.Set("with_labels_details", "true")

	sdk.LogDebug(qc.Logger, "work issues", "project", project.Name, "project_ref_id", project.RefID, "params", params)

	objectPath := sdk.JoinURL("projects", url.QueryEscape(project.RefID), "issues")

	var rawissues []IssueModel

	pi, err = qc.Get(objectPath, params, &rawissues)
	if err != nil {
		return
	}

	projectID := sdk.NewWorkProjectID(qc.CustomerID, project.RefID, "gitlab")

	sdk.LogDebug(qc.Logger, "issues found", "len", len(rawissues))

	for _, rawissue := range rawissues {
		if rawissue.UpdatedAt.Before(stopOnUpdatedAt) {
			return
		}

		issueRefID := strconv.FormatInt(rawissue.ID, 10)
		issueID := sdk.NewWorkIssueID(qc.CustomerID, issueRefID, qc.RefType)

		item := sdk.WorkIssue{}
		item.Active = true
		item.CustomerID = qc.CustomerID
		item.RefType = qc.RefType
		item.RefID = issueRefID

		if rawissue.Assignee != nil {
			item.AssigneeRefID = fmt.Sprint(rawissue.Assignee.ID)
		}

		item.ReporterRefID = fmt.Sprint(rawissue.Author.ID)
		item.CreatorRefID = fmt.Sprint(rawissue.Author.ID)
		item.Description = rawissue.Description
		if rawissue.Epic != nil {
			epicID := sdk.NewWorkIssueID(qc.CustomerID, strconv.FormatInt(rawissue.Epic.ID, 10), qc.RefType)
			item.EpicID = sdk.StringPointer(epicID)
			item.ParentID = epicID
		}
		item.Identifier = rawissue.References.Full
		item.ProjectID = sdk.NewWorkProjectID(qc.CustomerID, project.RefID, qc.RefType)
		item.Title = rawissue.Title
		item.Status = rawissue.State
		if item.Status == "opened" {
			item.StatusID = sdk.NewWorkIssueStatusID(qc.CustomerID, "gitlab", "1")
		} else {
			item.StatusID = sdk.NewWorkIssueStatusID(qc.CustomerID, "gitlab", "2")
		}
		// priority
		// due_date

		tags := make([]string, 0)
		for _, label := range rawissue.Labels {
			tags = append(tags, label.Name)
		}

		qc.WorkManager.AddIssue(issueID, rawissue.State == "opened", projectID, rawissue.Labels, rawissue.Milestone, rawissue.Assignee, rawissue.Weight)

		item.Tags = tags
		item.Type = "Bug"
		item.URL = rawissue.WebURL

		sdk.ConvertTimeToDateModel(rawissue.CreatedAt, &item.CreatedDate)
		sdk.ConvertTimeToDateModel(rawissue.UpdatedAt, &item.UpdatedDate)

		if rawissue.Milestone != nil {
			item.SprintIds = []string{sdk.NewAgileSprintID(qc.CustomerID, strconv.FormatInt(int64(rawissue.Milestone.RefID), 10), qc.RefType)}

			duedate, err := time.Parse("2006-01-02", rawissue.Milestone.DueDate)
			if err != nil {
				duedate = time.Time{}
			}
			sdk.ConvertTimeToDateModel(duedate, &item.PlannedEndDate)

			startdate, err := time.Parse("2006-01-02", rawissue.Milestone.StartDate)
			if err != nil {
				startdate = time.Time{}
			}
			sdk.ConvertTimeToDateModel(startdate, &item.PlannedStartDate)
		}

		issues <- item
	}

	return
}

type IssueWebHook struct {
	IID int64 `json:"iid"`
}

type IssueModel struct {
	ID                 int64     `json:"id"`
	Iid                int       `json:"iid"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	State              string    `json:"state"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	Labels             []*Label  `json:"labels"`
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
	MovedToID interface{} `json:"moved_to_id"`
	Epic      *struct {
		ID int64 `json:"id"`
	} `json:"epic"`
	Weight       *int  `json:"weight"`
	ProjectRefID int64 `json:"project_id"`
}

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
