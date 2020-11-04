package internal

import (
	"time"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

const projectCapabilityCacheKeyPrefix = "project_capability_"

func (ge *GitlabExport) writeProjectCapacity(repo *sdk.WorkProject) error {
	var cacheKey = projectCapabilityCacheKeyPrefix + repo.ID
	if !ge.historical && ge.state.Exists(cacheKey) {
		return nil
	}
	sdk.LogDebug(ge.logger, "debug-debug calculating capabilities", "repo", repo.Name)
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
	capability.IssueMutationFields = createMutationFields()
	ge.state.SetWithExpires(cacheKey, 1, time.Hour*24*30)
	sdk.LogDebug(ge.logger, "debug-debug finished ", "repo", repo.Name, "capabilities", sdk.Stringify(capability))
	return ge.pipe.Write(&capability)
}

func createMutationFields() []sdk.WorkProjectCapabilityIssueMutationFields {

	issueTypes := []string{
		api.BugIssueType,
		api.IncidentIssueType,
		api.EnhancementIssueType,
		api.MilestoneIssueType,
		api.EpicIssueType,
	}

	return []sdk.WorkProjectCapabilityIssueMutationFields{
		{
			AlwaysAvailable:   true,
			Name:              "Title",
			Description:       sdk.StringPointer("title of the issue"),
			AlwaysRequired:    true,
			RefID:             "title",
			Immutable:         false,
			Type:              sdk.WorkProjectCapabilityIssueMutationFieldsTypeString,
			AvailableForTypes: issueTypes,
			RequiredByTypes:   issueTypes,
		}, {
			AlwaysAvailable:   true,
			Name:              "Description",
			Description:       sdk.StringPointer("description of the issue"),
			AlwaysRequired:    true,
			RefID:             "description",
			Immutable:         false,
			Type:              sdk.WorkProjectCapabilityIssueMutationFieldsTypeTextbox,
			AvailableForTypes: issueTypes,
			RequiredByTypes:   issueTypes,
		}, {
			AlwaysAvailable:   true,
			Name:              "IssueType",
			Description:       sdk.StringPointer("issue type"),
			AlwaysRequired:    true,
			RefID:             "issueType",
			Immutable:         false,
			AvailableForTypes: issueTypes,
			RequiredByTypes:   issueTypes,
			Type:              sdk.WorkProjectCapabilityIssueMutationFieldsTypeWorkIssueType,
		},
		{
			AlwaysAvailable: false,
			Name:            "Assignee",
			Description:     sdk.StringPointer("assigne"),
			AlwaysRequired:  false,
			RefID:           "assignee",
			Immutable:       false,
			AvailableForTypes: append([]string{
				api.BugIssueType,
				api.IncidentIssueType,
				api.EnhancementIssueType,
			}),
			Type: sdk.WorkProjectCapabilityIssueMutationFieldsTypeUser,
		},
		{
			AlwaysAvailable: false,
			Name:            "Epic",
			Description:     sdk.StringPointer("epic"),
			AlwaysRequired:  false,
			RefID:           "epic",
			Immutable:       false,
			AvailableForTypes: append([]string{
				api.BugIssueType,
				api.IncidentIssueType,
				api.EnhancementIssueType,
			}),
			Type: sdk.WorkProjectCapabilityIssueMutationFieldsTypeEpic,
		},
		{
			AlwaysAvailable: false,
			Name:            "Milestone",
			Description:     sdk.StringPointer("milestone"),
			AlwaysRequired:  false,
			RefID:           "milestone",
			Immutable:       false,
			AvailableForTypes: append([]string{
				api.BugIssueType,
				api.IncidentIssueType,
				api.EnhancementIssueType,
			}),
			Type: sdk.WorkProjectCapabilityIssueMutationFieldsTypeStringArray,
		},
		{
			AlwaysAvailable: false,
			Name:            "Weight",
			Description:     sdk.StringPointer("weight"),
			AlwaysRequired:  false,
			RefID:           "weight",
			Immutable:       false,
			AvailableForTypes: append([]string{
				api.BugIssueType,
				api.IncidentIssueType,
				api.EnhancementIssueType,
			}),
			Type: sdk.WorkProjectCapabilityIssueMutationFieldsTypeNumber,
		},
		{
			AlwaysAvailable: false,
			Name:            "Due Date",
			Description:     sdk.StringPointer("due date"),
			AlwaysRequired:  false,
			RefID:           "dueDate",
			Immutable:       false,
			AvailableForTypes: append([]string{
				api.BugIssueType,
				api.IncidentIssueType,
				api.EnhancementIssueType,
				api.EpicIssueType,
				api.IncidentIssueType,
			}),
			Type: sdk.WorkProjectCapabilityIssueMutationFieldsTypeDate,
		},
		{
			AlwaysAvailable: false,
			Name:            "Start Date",
			Description:     sdk.StringPointer("start date"),
			AlwaysRequired:  false,
			RefID:           "startDate",
			Immutable:       false,
			AvailableForTypes: append([]string{
				api.MilestoneIssueType,
				api.EpicIssueType,
			}),
			Type: sdk.WorkProjectCapabilityIssueMutationFieldsTypeDate,
		},
		// {
		// 	AlwaysAvailable: false,
		// 	Name:            "Label",
		// 	Description:     sdk.StringPointer("label"),
		// 	AlwaysRequired:  false,
		// 	RefID:           "milestone",
		// 	Immutable:       false,
		// 	AvailableForTypes: append([]string{
		// 		api.BugIssueType,
		// 		api.IncidentIssueType,
		// 		api.EnhancementIssueType,
		// 	}),
		// 	Type: sdk.WorkProjectCapabilityIssueMutationFieldsTypeStringArray,
		// },
		// We may need some a new type of this to be able to select 0 or many
	}

}
