package internal

import (
	"sync"

	"github.com/pinpt/agent.next/sdk"
)

// IssueManager is a manager for users
// easyjson:skip
type IssueManager struct {
	refLabelsIssues       sync.Map
	refIssuesState        sync.Map
	refIssuesProject      sync.Map
	refMilestoneIssuesIds sync.Map
	refOpenIssuesProject  sync.Map
	refCloseIssuesProject sync.Map
	logger                sdk.Logger
}

// AddIssueID add issueid
func (i *IssueManager) AddIssueID(issueID string, issueState string, projectID string, milestoneRefID int64, labelsRefIDs []int64) {

	sdk.LogDebug(i.logger, "check-1", "projectID", projectID, "issuesID", issueID, "labels", labelsRefIDs)
	for _, labelRefID := range labelsRefIDs {
		issuesByLabel, ok := i.refLabelsIssues.Load(labelRefID)
		if !ok {
			sdk.LogError(i.logger, "adding issue to label1", "label", labelRefID, "issue", issueID)
			i.refLabelsIssues.Store(labelRefID, []string{issueID})
		} else {
			sdk.LogError(i.logger, "adding issue to label2", "label", labelRefID, "issue", issueID)
			issuesByLbl := issuesByLabel.([]string)
			issuesByLbl = append(issuesByLbl, issueID)

			i.refLabelsIssues.Store(labelRefID, issuesByLbl)
		}
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

	// // open issues - by project
	// sdk.LogDebug(i.logger, "check1", "projectID", projectID, "issuesID", issueID)
	// if issueState == "open" {
	// 	openIssuesByProject, ok := i.refOpenIssuesProject.Load(projectID)
	// 	if !ok {
	// 		i.refOpenIssuesProject.Store(projectID, []string{issueID})
	// 		sdk.LogDebug(i.logger, "check2", "projectID", projectID, "issuesID", issueID)
	// 	} else {
	// 		openIssues := openIssuesByProject.([]string)
	// 		openIssues = append(openIssues, issueID)
	// 		sdk.LogDebug(i.logger, "check3", "projectID", projectID, "issuesID", issueID)
	// 		i.refOpenIssuesProject.Store(projectID, openIssues)
	// 	}
	// }

	// // open issues - by project
	// if issueState == "closed" {
	// 	closedIssuesByProject, ok := i.refCloseIssuesProject.Load(projectID)
	// 	if !ok {
	// 		i.refCloseIssuesProject.Store(projectID, []string{issueID})
	// 	} else {
	// 		closeIssues := closedIssuesByProject.([]string)
	// 		closeIssues = append(closeIssues, issueID)
	// 		i.refCloseIssuesProject.Store(projectID, closeIssues)
	// 	}
	// }

	sdk.LogDebug(i.logger, "check-3", "projectID", projectID, "issuesID", issueID)
	// milestone - issuesIDs
	issueSt, ok := i.refIssuesState.Load(issueID)
	if !ok {
		i.refIssuesState.Store(issueID, issueState)
	} else {
		i.refIssuesState.Store(issueID, issueSt.(string))
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
func (i *IssueManager) GetIssuesIDsByLabelRefIDMilestonRefID(labelRefID int64, milestoneRefID int64) []string {

	issuesByLabel, ok := i.refLabelsIssues.Load(labelRefID)
	if !ok {
		return []string{}
	}

	issuesByLabelMap := make(map[string]bool)
	for _, issue := range issuesByLabel.([]string) {
		issuesByLabelMap[issue] = false
	}

	issuesByMilestone := i.GetIssuesIDsByMilestone(milestoneRefID)
	for _, issue := range issuesByMilestone {
		issuesByLabelMap[issue] = true
	}

	issues := make([]string, 0)
	for issue := range issuesByLabelMap {
		issues = append(issues, issue)
	}

	return issues
}

// GetProjectIDsByLabel get project ids by label
// func (i *IssueManager) GetProjectIDsByLabel(labelID int64) map[int64]bool {
// 	lblInfo, ok := i.refLabelsIssues.Load(labelID)
// 	if !ok {
// 		return make(map[int64]bool)
// 	}

// 	return lblInfo.(labelInfo).projectIDs
// }

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
func (i *IssueManager) GetOpenIssuesIDsByProject(projectLabels []int64, projectID string) []string {

	openIssues := make([]string, 0)

	issuesByProjectMap, ok := i.refIssuesProject.Load(projectID)

	copyIssuesByProject := make(map[string]bool)

	sdk.LogError(i.logger, "issues-map", "issues", issuesByProjectMap, "projectID", projectID, "projectlabels", projectLabels)

	if ok {
		for issueID := range issuesByProjectMap.(map[string]bool) {
			copyIssuesByProject[issueID] = true
		}
		// exclude all issues belonging to a project label
		for _, projectLabel := range projectLabels {
			issuesByLabel, ok := i.refLabelsIssues.Load(projectLabel)
			// sdk.LogError(i.logger, "issues by label", "label", projectLabel, "issues", issuesByLabel)
			if !ok {
				continue
			}
			sdk.LogDebug(i.logger, "issues by label", "label", projectLabel, "issues", issuesByLabel)
			for _, issue := range issuesByLabel.([]string) {
				copyIssuesByProject[issue] = false
			}

		}

		for issue, issueValue := range copyIssuesByProject {

			issueState, _ := i.refIssuesState.Load(issue)

			sdk.LogDebug(i.logger, "issue state", "issue", issue, "issueValue", issueValue, "issueState", issueState.(string))

			if issueValue && issueState.(string) == "opened" {
				openIssues = append(openIssues, issue)
			}

		}
	}

	sdk.LogDebug(i.logger, "returning open issues", "isues", openIssues)

	return openIssues
}

// GetCloseIssuesIDsByProject get open issues ids by project
func (i *IssueManager) GetCloseIssuesIDsByProject2(projectLabels []int64, projectID string) []string {

	closedIssues := make([]string, 0)
	copyIssuesByProject := make(map[string]bool)

	issuesByProjectMap, ok := i.refIssuesProject.Load(projectID)

	if ok {
		for issueID := range issuesByProjectMap.(map[string]bool) {
			copyIssuesByProject[issueID] = true
		}
		// exclude all issues belonging to a project label
		for _, projectLabel := range projectLabels {
			issuesByLabel, ok := i.refLabelsIssues.Load(projectLabel)
			if !ok {
				continue
			}
			for _, issue := range issuesByLabel.([]string) {
				copyIssuesByProject[issue] = false
			}

		}

		for issue, issueValue := range copyIssuesByProject {

			issueState, _ := i.refIssuesState.Load(issue)

			if issueValue && issueState.(string) == "closed" {
				closedIssues = append(closedIssues, issue)
			}

		}
	}

	return closedIssues
}

// GetOpenIssuesIDsByGroupBoardLabels get open issues ids by project
func (i *IssueManager) GetOpenIssuesIDsByGroupBoardLabels(boardLabels []int64, projectIDs []string) []string {

	openIssues := make([]string, 0)
	copyIssuesByProject := make(map[string]bool)

	for _, projectID := range projectIDs {
		issuesByProjectMap, ok := i.refIssuesProject.Load(projectID)

		sdk.LogError(i.logger, "issues-map", "issues", issuesByProjectMap, "projectID", projectID, "boardLabels", boardLabels)

		if ok {
			for issueID := range issuesByProjectMap.(map[string]bool) {
				copyIssuesByProject[issueID] = true
			}
		}
	}

	// exclude all issues belonging to a project label
	for _, projectLabel := range boardLabels {
		issuesByLabel, ok := i.refLabelsIssues.Load(projectLabel)
		// sdk.LogError(i.logger, "issues by label", "label", projectLabel, "issues", issuesByLabel)
		if !ok {
			continue
		}
		sdk.LogDebug(i.logger, "issues by label", "label", projectLabel, "issues", issuesByLabel)
		for _, issue := range issuesByLabel.([]string) {
			copyIssuesByProject[issue] = false
		}

	}

	for issue, issueValue := range copyIssuesByProject {

		issueState, _ := i.refIssuesState.Load(issue)

		sdk.LogDebug(i.logger, "issue state", "issue", issue, "issueValue", issueValue, "issueState", issueState.(string))

		if issueValue && issueState.(string) == "opened" {
			openIssues = append(openIssues, issue)
		}

	}

	sdk.LogDebug(i.logger, "returning open issues", "isues", openIssues)

	return openIssues
}

func (i *IssueManager) GetClosedIssuesIDsByGroupBoardLabels(boardLabels []int64, projectIDs []string) []string {

	closedIssues := make([]string, 0)
	copyIssuesByProject := make(map[string]bool)

	for _, projectID := range projectIDs {
		issuesByProjectMap, ok := i.refIssuesProject.Load(projectID)

		sdk.LogError(i.logger, "issues-map", "issues", issuesByProjectMap, "projectID", projectID, "boardLabels", boardLabels)

		if ok {
			for issueID := range issuesByProjectMap.(map[string]bool) {
				copyIssuesByProject[issueID] = true
			}
		}
	}

	// exclude all issues belonging to a project label
	for _, projectLabel := range boardLabels {
		issuesByLabel, ok := i.refLabelsIssues.Load(projectLabel)
		// sdk.LogError(i.logger, "issues by label", "label", projectLabel, "issues", issuesByLabel)
		if !ok {
			continue
		}
		sdk.LogDebug(i.logger, "issues by label", "label", projectLabel, "issues", issuesByLabel)
		for _, issue := range issuesByLabel.([]string) {
			copyIssuesByProject[issue] = false
		}

	}

	for issue, issueValue := range copyIssuesByProject {

		issueState, _ := i.refIssuesState.Load(issue)

		sdk.LogDebug(i.logger, "issue state", "issue", issue, "issueValue", issueValue, "issueState", issueState.(string))

		if issueValue && issueState.(string) == "closed" {
			closedIssues = append(closedIssues, issue)
		}

	}

	sdk.LogDebug(i.logger, "returning closed issues", "isues", closedIssues)

	return closedIssues
}

// NewIssueManager returns a new instance
func NewIssueManager(logger sdk.Logger) *IssueManager {
	return &IssueManager{
		logger: logger,
	}
}
