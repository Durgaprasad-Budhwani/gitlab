package internal

import (
	"sync"

	"github.com/pinpt/agent.next/sdk"
)

// IssueManager is a manager for users
// easyjson:skip
type IssueManager struct {
	refLabelsIssues       sync.Map
	refIssuesProject      sync.Map
	refMilestoneIssuesIds sync.Map
	refOpenIssuesProject  sync.Map
	refCloseIssuesProject sync.Map
	logger                sdk.Logger
}

type labelInfo struct {
	issueIDs   []string
	projectIDs map[int64]bool
}

// AddIssueID add issueid
func (i *IssueManager) AddIssueID(labelID int64, issueID string, projectID string, milestoneRefID int64) {

	sdk.LogDebug(i.logger, "check-1", "projectID", projectID, "issuesID", issueID)
	lblInfo, ok := i.refLabelsIssues.Load(labelID)
	if !ok {
		i.refLabelsIssues.Store(labelID, labelInfo{issueIDs: []string{issueID}, projectIDs: map[int64]bool{labelID: true}})
	} else {
		labelInfo := lblInfo.(labelInfo)

		labelInfo.issueIDs = append(labelInfo.issueIDs, issueID)
		labelInfo.projectIDs[labelID] = true

		i.refLabelsIssues.Store(labelID, labelInfo)
	}

	sdk.LogDebug(i.logger, "check-2", "projectID", projectID, "issuesID", issueID)
	// project issues
	prjIssues, ok := i.refIssuesProject.Load(projectID)
	if !ok {
		i.refIssuesProject.Store(projectID, map[string]bool{issueID: true})
	} else {
		projectIssues := prjIssues.(map[string]bool)
		projectIssues[issueID] = true
		i.refIssuesProject.Store(projectID, projectIssues)
	}

	sdk.LogDebug(i.logger, "check-3", "projectID", projectID, "issuesID", issueID)
	// milestone - issuesIDs
	milestoneIssuesIDs, ok := i.refMilestoneIssuesIds.Load(milestoneRefID)
	if !ok {
		i.refMilestoneIssuesIds.Store(milestoneRefID, []string{issueID})
	} else {
		milestoneIssues := milestoneIssuesIDs.([]string)
		milestoneIssues = append(milestoneIssues, issueID)
		i.refMilestoneIssuesIds.Store(milestoneRefID, milestoneIssues)
	}

	// open issues - by project
	sdk.LogDebug(i.logger, "check1", "projectID", projectID, "issuesID", issueID)
	if labelID == 0 {
		openIssuesByProject, ok := i.refOpenIssuesProject.Load(projectID)
		if !ok {
			i.refOpenIssuesProject.Store(projectID, []string{issueID})
			sdk.LogDebug(i.logger, "check2", "projectID", projectID, "issuesID", issueID)
		} else {
			openIssues := openIssuesByProject.([]string)
			openIssues = append(openIssues, issueID)
			sdk.LogDebug(i.logger, "check3", "projectID", projectID, "issuesID", issueID)
			i.refOpenIssuesProject.Store(projectID, openIssues)
		}
	}

	// open issues - by project
	if labelID == 1 {
		closedIssuesByProject, ok := i.refCloseIssuesProject.Load(projectID)
		if !ok {
			i.refCloseIssuesProject.Store(projectID, []string{issueID})
		} else {
			closeIssues := closedIssuesByProject.([]string)
			closeIssues = append(closeIssues, issueID)
			i.refCloseIssuesProject.Store(projectID, closeIssues)
		}
	}
}

// GetIssuesIDsByMilestone get issues ids by milestone
func (i *IssueManager) GetIssuesIDsByMilestone(milestoneRefID int64) []string {
	issues, ok := i.refMilestoneIssuesIds.Load(milestoneRefID)
	if !ok {
		return []string{}
	}
	return issues.([]string)
}

// GetIssuesIDsByLabelID get issues ids by labelID
func (i *IssueManager) GetIssuesIDsByLabelID(labelID int64) []string {
	lblInfo, ok := i.refLabelsIssues.Load(labelID)
	if !ok {
		return []string{}
	}
	return lblInfo.(labelInfo).issueIDs
}

// GetProjectIDsByLabel get project ids by label
func (i *IssueManager) GetProjectIDsByLabel(labelID int64) map[int64]bool {
	lblInfo, ok := i.refLabelsIssues.Load(labelID)
	if !ok {
		return make(map[int64]bool)
	}

	return lblInfo.(labelInfo).projectIDs
}

// GetIssuesIDsByProject get issues ids by project
func (i *IssueManager) GetIssuesIDsByProject(projectID string) []string {
	issues, ok := i.refIssuesProject.Load(projectID)
	if !ok {
		return make([]string, 0)
	}

	var issuesIDs []string
	for issueID := range issues.(map[string]bool) {
		issuesIDs = append(issuesIDs, issueID)
	}

	return issuesIDs
}

// GetOpenIssuesIDsByProject get open issues ids by project
func (i *IssueManager) GetOpenIssuesIDsByProject(projectID string) []string {
	issuesIDs, ok := i.refOpenIssuesProject.Load(projectID)
	if !ok {
		return make([]string, 0)
	}

	sdk.LogDebug(i.logger, "check4", "projectID", projectID, "issuesIDs", issuesIDs)

	return issuesIDs.([]string)
}

// GetCloseIssuesIDsByProject get open issues ids by project
func (i *IssueManager) GetCloseIssuesIDsByProject(projectID string) []string {
	issuesIDs, ok := i.refCloseIssuesProject.Load(projectID)
	if !ok {
		return make([]string, 0)
	}

	return issuesIDs.([]string)
}

// NewIssueManager returns a new instance
func NewIssueManager(logger sdk.Logger) *IssueManager {
	return &IssueManager{
		logger: logger,
	}
}
