package internal

import (
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

const projectCapabilityCacheKeyPrefix = "project_capability_"

func (ge *GitlabExport) writeProjectCapacity(repo *sdk.WorkProject) error {
	var cacheKey = projectCapabilityCacheKeyPrefix + repo.ID
	if !ge.historical && ge.state.Exists(cacheKey) {
		return nil
	}
	var capability sdk.WorkProjectCapability
	capability.CustomerID = repo.CustomerID
	capability.RefID = repo.RefID
	capability.RefType = repo.RefType
	capability.IntegrationInstanceID = repo.IntegrationInstanceID
	capability.ProjectID = sdk.NewWorkProjectID(repo.CustomerID, repo.RefID, ge.qc.RefType)
	capability.UpdatedAt = repo.UpdatedAt
	capability.Attachments = false // TODO
	capability.ChangeLogs = true
	capability.DueDates = false
	capability.Epics = false // PENDING
	capability.InProgressStates = false
	capability.KanbanBoards = true
	capability.LinkedIssues = false // TODO
	capability.Parents = false      // TODO
	capability.Priorities = false
	capability.Resolutions = false
	capability.Sprints = true
	capability.StoryPoints = false // TODO could this be equal to weight?
	ge.state.SetWithExpires(cacheKey, 1, time.Hour*24*30)
	return ge.pipe.Write(&capability)
}
