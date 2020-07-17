package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pinpt/agent.next/sdk"
	"github.com/pinpt/go-common/v10/datetime"
	pstrings "github.com/pinpt/go-common/v10/strings"
)

func WorkIssuesPage(
	qc QueryContext,
	project *sdk.WorkProject,
	params url.Values,
	issues chan sdk.WorkIssue) (pi NextPage, err error) {

	params.Set("scope", "all")

	sdk.LogDebug(qc.Logger, "work issues", "project", project.Name, "project_ref_id", project.RefID, "params", params)

	objectPath := pstrings.JoinURL("projects", url.QueryEscape(project.RefID), "issues")

	var rawissues []IssueModel

	pi, err = qc.Get(objectPath, params, &rawissues)
	if err != nil {
		return
	}
	for _, rawissue := range rawissues {

		idparts := strings.Split(project.RefID, "/")
		var identifier string
		if len(idparts) == 1 {
			identifier = idparts[0] + "-" + fmt.Sprint(rawissue.Iid)
		} else {
			identifier = idparts[1] + "-" + fmt.Sprint(rawissue.Iid)
		}
		item := sdk.WorkIssue{}
		item.CustomerID = qc.CustomerID
		item.RefType = qc.RefType
		item.RefID = fmt.Sprint(rawissue.Iid)

		item.AssigneeRefID = fmt.Sprint(rawissue.Assignee.ID)
		item.ReporterRefID = fmt.Sprint(rawissue.Author.ID)
		item.CreatorRefID = fmt.Sprint(rawissue.Author.ID)
		item.Description = rawissue.Description
		if rawissue.EpicIid != 0 {
			item.EpicID = pstrings.Pointer(fmt.Sprint(rawissue.EpicIid))
		}
		item.Identifier = identifier
		item.ProjectID = sdk.NewWorkProjectID(qc.CustomerID, project.RefID, qc.RefType)
		item.Title = rawissue.Title
		item.Status = rawissue.State
		item.Tags = rawissue.Labels
		item.Type = "Issue"
		item.URL = rawissue.WebURL

		datetime.ConvertToModel(rawissue.CreatedAt, &item.CreatedDate)
		datetime.ConvertToModel(rawissue.UpdatedAt, &item.UpdatedDate)

		item.SprintIds = []string{sdk.NewAgileSprintID(qc.CustomerID, strconv.FormatInt(int64(rawissue.Milestone.Iid), 10), qc.RefType)}
		duedate, err := time.Parse("2006-01-02", rawissue.Milestone.DueDate)
		if err != nil {
			duedate = time.Time{}
		}
		datetime.ConvertToModel(duedate, &item.PlannedEndDate)

		startdate, err := time.Parse("2006-01-02", rawissue.Milestone.StartDate)
		if err != nil {
			startdate = time.Time{}
		}
		datetime.ConvertToModel(startdate, &item.PlannedStartDate)

		issues <- item
	}

	return
}

type IssueModel struct {
	ID          int       `json:"id"`
	Iid         int       `json:"iid"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	State       string    `json:"state"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Labels      []string  `json:"labels"`
	Milestone   struct {
		ID          int       `json:"id"`
		Iid         int       `json:"iid"`
		GroupID     int       `json:"group_id"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		State       string    `json:"state"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		DueDate     string    `json:"due_date"`
		StartDate   string    `json:"start_date"`
		WebURL      string    `json:"web_url"`
	} `json:"milestone"`
	Assignees          []UserModel `json:"assignees"`
	Author             UserModel   `json:"author"`
	Assignee           UserModel   `json:"assignee"`
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
	Weight   interface{} `json:"weight"`
	HasTasks bool        `json:"has_tasks"`
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
	EpicIid   int         `json:"epic_iid"`
	Epic      interface{} `json:"epic"`
}

type UserModel struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	ID        int64  `json:"id"`
	AvatarURL string `json:"avatar_url"`
	WebURL    string `json:"web_url"`
}
