package api

import (
	"fmt"
	"net/url"
	"time"

	"github.com/pinpt/agent.next/sdk"
	"github.com/pinpt/go-common/v10/datetime"
	pstrings "github.com/pinpt/go-common/v10/strings"
)

func WorkSprintPage(qc QueryContext, project *sdk.WorkProject, params url.Values) (pi NextPage, res []*sdk.AgileSprint, err error) {

	sdk.LogDebug(qc.Logger, "work sprints", "project", project.Name, "project_ref_id", project.RefID, "params", params)

	objectPath := pstrings.JoinURL("projects", url.QueryEscape(project.RefID), "milestones")
	var rawsprints []struct {
		ID          int       `json:"id"`
		Iid         int       `json:"iid"`
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
	pi, err = qc.Request(objectPath, params, &rawsprints)
	if err != nil {
		return
	}
	for _, rawsprint := range rawsprints {

		item := &sdk.AgileSprint{}
		item.CustomerID = qc.CustomerID
		item.RefType = qc.RefType
		item.RefID = fmt.Sprint(rawsprint.Iid)

		start, err := time.Parse("2006-01-02", rawsprint.StartDate)
		if err == nil {
			datetime.ConvertToModel(start, &item.StartedDate)
		} else {
			if rawsprint.StartDate != "" {
				sdk.LogError(qc.Logger, "could not figure out start date, skipping sprint object", "err", err, "start_date", rawsprint.StartDate)
				continue
			}
		}
		end, err := time.Parse("2006-01-02", rawsprint.DueDate)
		if err == nil {
			datetime.ConvertToModel(end, &item.EndedDate)
		} else {
			if rawsprint.DueDate != "" {
				sdk.LogError(qc.Logger, "could not figure out due date, skipping sprint object", "err", err, "due_date", rawsprint.DueDate)
				continue
			}
		}

		if rawsprint.State == "closed" {
			datetime.ConvertToModel(rawsprint.UpdatedAt, &item.CompletedDate)
			item.Status = sdk.AgileSprintStatusClosed
		} else {
			if !start.IsZero() && start.After(time.Now()) {
				item.Status = sdk.AgileSprintStatusFuture
			} else {
				item.Status = sdk.AgileSprintStatusActive
			}
		}
		item.Goal = rawsprint.Description
		item.Name = rawsprint.Title
		item.RefID = fmt.Sprint(rawsprint.ID)
		item.RefType = qc.RefType

		res = append(res, item)
	}

	return
}
