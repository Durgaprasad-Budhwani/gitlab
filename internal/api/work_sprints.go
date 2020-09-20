package api

import (
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/agent.next/sdk"
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

func RepoSprintsPage(qc QueryContext, project *sdk.SourceCodeRepo, params url.Values) (pi NextPage, res []*sdk.AgileSprint, err error) {

	sdk.LogDebug(qc.Logger, "project work sprints", "project", project.Name, "project_ref_id", project.RefID, "params", params)

	objectPath := sdk.JoinURL("projects", url.QueryEscape(project.RefID), "milestones")

	return CommonSprintsPage(qc, params, objectPath)
}

func GroupSprintsPage(qc QueryContext, namespace *Namespace, params url.Values) (pi NextPage, res []*sdk.AgileSprint, err error) {

	sdk.LogDebug(qc.Logger, "group work sprints", "group", namespace.Name, "group_ref_id", namespace.ID, "params", params)

	objectPath := sdk.JoinURL("groups", url.QueryEscape(namespace.ID), "milestones")

	return CommonSprintsPage(qc, params, objectPath)
}

func CommonSprintsPage(qc QueryContext, params url.Values, url string) (pi NextPage, res []*sdk.AgileSprint, err error) {

	var rawsprints []Milestone
	pi, err = qc.Get(url, params, &rawsprints)
	if err != nil {
		return
	}
	for _, rawsprint := range rawsprints {

		sdk.LogDebug(qc.Logger, "debug-debug", "sprint-ref-id", rawsprint.RefID, "sprint", rawsprint)
		qc.WorkManager.AddMilestoneDetails(rawsprint.RefID, rawsprint)

		sprintRefID := strconv.FormatInt(rawsprint.RefID, 10)

		sprint := &sdk.AgileSprint{}
		sprint.ID = sdk.NewAgileSprintID(qc.CustomerID, sprintRefID, qc.RefType)
		sprint.Active = true
		sprint.CustomerID = qc.CustomerID
		sprint.RefType = qc.RefType
		sprint.RefID = sprintRefID

		start, err := time.Parse("2006-01-02", rawsprint.StartDate)
		if err == nil {
			sdk.ConvertTimeToDateModel(start, &sprint.StartedDate)
		} else {
			if rawsprint.StartDate != "" {
				sdk.LogError(qc.Logger, "could not figure out start date, skipping sprint object", "err", err, "start_date", rawsprint.StartDate)
				continue
			}
		}
		end, err := time.Parse("2006-01-02", rawsprint.DueDate)
		if err == nil {
			sdk.ConvertTimeToDateModel(end, &sprint.EndedDate)
		} else {
			if rawsprint.DueDate != "" {
				sdk.LogError(qc.Logger, "could not figure out due date, skipping sprint object", "err", err, "due_date", rawsprint.DueDate)
				continue
			}
		}

		if rawsprint.State == "closed" {
			sdk.ConvertTimeToDateModel(rawsprint.UpdatedAt, &sprint.CompletedDate)
			sprint.Status = sdk.AgileSprintStatusClosed
		} else {
			if !start.IsZero() && start.After(time.Now()) {
				sprint.Status = sdk.AgileSprintStatusFuture
			} else {
				sprint.Status = sdk.AgileSprintStatusActive
			}
		}

		sprint.Goal = rawsprint.Description
		sprint.Name = rawsprint.Title
		sprint.URL = sdk.StringPointer(rawsprint.WebURL)

		res = append(res, sprint)
	}

	return
}
