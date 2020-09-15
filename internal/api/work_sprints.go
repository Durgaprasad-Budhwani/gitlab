package api

import (
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/agent.next/sdk"
)

func SprintsPage(qc QueryContext, project *sdk.SourceCodeRepo, params url.Values) (pi NextPage, res []*sdk.AgileSprint, err error) {

	sdk.LogDebug(qc.Logger, "work sprints", "project", project.Name, "project_ref_id", project.RefID, "params", params)

	objectPath := sdk.JoinURL("projects", url.QueryEscape(project.RefID), "milestones")
	var rawsprints []struct {
		ID          int64     `json:"id"`
		ProjectID   int       `json:"project_id"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		State       string    `json:"state"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		DueDate     string    `json:"due_date"`
		StartDate   string    `json:"start_date"`
		WebURL      string    `json:"web_url"`
	}
	pi, err = qc.Get(objectPath, params, &rawsprints)
	if err != nil {
		return
	}
	for _, rawsprint := range rawsprints {

		sprintRefID := strconv.FormatInt(rawsprint.ID, 10)

		sprint := &sdk.AgileSprint{}
		sprint.ID = sdk.NewAgileSprintID(qc.CustomerID, sprintRefID, qc.RefType)
		sprint.Active = true
		sprint.CustomerID = qc.CustomerID
		sprint.RefType = qc.RefType
		sprint.RefID = sprintRefID
		sprint.BoardID = sdk.StringPointer(qc.SprintManager.GetBoardID(rawsprint.ID))

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

		sprint.ProjectIds = []string{sdk.NewWorkProjectID(qc.CustomerID, project.RefID, qc.RefType)}
		sprint.IssueIds = qc.IssueManager.GetIssuesIDsByMilestone(rawsprint.ID)
		sprint.Columns = qc.SprintManager.GetSprintColumnsIssuesIDs(sprintRefID)

		sprint.Goal = rawsprint.Description
		sprint.Name = rawsprint.Title
		sprint.URL = sdk.StringPointer(rawsprint.WebURL)

		res = append(res, sprint)
	}

	return
}
