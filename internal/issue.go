package internal

import (
	"sync"

	"github.com/pinpt/agent.next/sdk"
)

// IssueManager is a manager for users
// easyjson:skip
type IssueManager struct {
	refLabelsIssues  sync.Map
	refIssuesProject sync.Map
}

type labelInfo struct {
	issueIDs   []string
	projectIDs map[int64]bool
}

// AddIssueID add issueid
func (i *IssueManager) AddIssueID(labelID int64, issueID string, projectID string) {
	lblInfo, ok := i.refLabelsIssues.Load(labelID)
	if !ok {
		i.refLabelsIssues.Store(labelID, labelInfo{issueIDs: make([]string, 0), projectIDs: make(map[int64]bool)})
	} else {
		labelInfo := lblInfo.(labelInfo)

		labelInfo.issueIDs = append(labelInfo.issueIDs, issueID)
		labelInfo.projectIDs[labelID] = true

		i.refLabelsIssues.Store(labelID, labelInfo)
	}

	// project issues
	prjIssues, ok := i.refIssuesProject.Load(projectID)
	if !ok {
		i.refIssuesProject.Store(labelID, []string{})
	} else {
		projectIssues := prjIssues.([]string)
		projectIssues = append(projectIssues, issueID)
		i.refIssuesProject.Store(labelID, projectIssues)
	}

}

// GetIssuesIDs get issues ids
func (i *IssueManager) GetIssuesIDs(labelID int64) []string {
	lblInfo, ok := i.refLabelsIssues.Load(labelID)
	if !ok {
		return []string{}
	}
	return lblInfo.(labelInfo).issueIDs
}

// GetProjectIDs get project issue
func (i *IssueManager) GetProjectIDs(labelID int64) map[int64]bool {
	lblInfo, ok := i.refLabelsIssues.Load(labelID)
	if !ok {
		return make(map[int64]bool)
	}

	return lblInfo.(labelInfo).projectIDs
}

// GetProjectIssuesIDs get project issues ids
func (i *IssueManager) GetProjectIssuesIDs(projectID string) []string {
	issues, ok := i.refLabelsIssues.Load(projectID)
	if !ok {
		return make([]string, 0)
	}

	return issues.([]string)
}

// NewIssueManager returns a new instance
func NewIssueManager(logger sdk.Logger) *IssueManager {
	return &IssueManager{}
}
