package internal

import (
	"strconv"
	"sync"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

// WorkManager work manager
type WorkManager struct {
	refProject           sync.Map
	refMilestonesDetails sync.Map
	logger               sdk.Logger
}

type issueDetail struct {
	Open           bool
	Labels         map[int64]*api.Label
	Assignee       *api.UserModel
	MilestoneRefID int64
	Weight         *int
}

type milestoneDetail struct {
	*api.Milestone
	boards map[string][]*api.Label
}

// AddIssue desc
func (w *WorkManager) AddIssue(issueID string, issueState bool, projectRefID int64, labels []*api.Label, milestone *api.Milestone, assignee *api.UserModel, weight *int) {

	var convertLabelsToMap = func(labels []*api.Label) map[int64]*api.Label {

		m := make(map[int64]*api.Label)

		for _, label := range labels {
			m[label.ID] = label
		}

		return m
	}

	var milestoneRefID int64
	if milestone != nil {
		milestoneRefID = milestone.RefID
	} else {
		milestoneRefID = 0
	}

	issueD := &issueDetail{
		Open:           issueState,
		Labels:         convertLabelsToMap(labels),
		Assignee:       assignee,
		MilestoneRefID: milestoneRefID,
		Weight:         weight,
	}

	projectRefIDStr := strconv.FormatInt(projectRefID, 10)

	projectIssues, ok := w.refProject.Load(projectRefIDStr)
	if !ok {
		sdk.LogDebug(w.logger, "debug-debug2-adding-issue-for-project", "projectID", projectRefID, "issueID", issueID)
		w.refProject.Store(projectRefIDStr, map[string]*issueDetail{issueID: issueD})
	} else {
		sdk.LogDebug(w.logger, "debug-debug2-adding-issue-for-project2", "projectID", projectRefID, "issueID", issueID)
		projectIssues := projectIssues.(map[string]*issueDetail)
		projectIssues[issueID] = issueD
		w.refProject.Store(projectRefIDStr, projectIssues)
	}

}

// GetBoardColumnIssues desc
func (w *WorkManager) GetBoardColumnIssues(projectsRefIDs []string, milestone *api.Milestone, boardLabels []*api.Label, boardLists []api.BoardList, columnLabel *api.Label, assignee *api.UserModel, weight *int) []string {

	issues := make([]string, 0)

	sdk.LogDebug(w.logger, "debug-debug2", "msg", "check1", "projectRefIDs", projectsRefIDs)

	var milestoneRefID int64
	if milestone != nil {
		milestoneRefID = milestone.RefID
	} else {
		milestoneRefID = 0
	}

	if columnLabel.ID != api.OpenColumn &&
		columnLabel.ID != api.ClosedColumn {
		boardLabels = append(boardLabels, columnLabel)
	}

	w.refProject.Range(func(k, v interface{}) bool {
		sdk.LogDebug(w.logger, "debug-debug2", "key", k.(string), "value", v.(map[string]*issueDetail))
		return true
	})

	// get project issues
	for _, projectRefID := range projectsRefIDs {
		sdk.LogDebug(w.logger, "debug-debug2", "msg", projectRefID)
		projectIssues, ok := w.refProject.Load(projectRefID)
		if !ok {
			sdk.LogDebug(w.logger, "debug-debug2-no-project-found", "project", projectRefID)
			continue
		}

		sdk.LogDebug(w.logger, "debug-debug22", "projectRefID", projectRefID, "issues", projectIssues)

		for issueID, issueDetails := range projectIssues.(map[string]*issueDetail) {

			sdk.LogDebug(w.logger, "debug-debug3", "issueID", issueID, "issueMilestoneRefID", issueDetails.MilestoneRefID, "milestoneRefID", milestoneRefID)

			cond1 := milestone == nil || issueDetails.MilestoneRefID == milestoneRefID
			cond2 := true
			if len(boardLabels) != 0 {
				for _, label := range boardLabels {
					if label.ID == api.OpenColumn || label.ID == api.ClosedColumn {
						continue
					}
					_, ok := issueDetails.Labels[label.ID]
					if !ok {
						cond2 = false
					}
				}
			}
			cond5 := true
			if columnLabel.ID == api.OpenColumn ||
				columnLabel.ID == api.ClosedColumn {
				for _, list := range boardLists {
					_, ok := issueDetails.Labels[list.Label.ID]
					if ok {
						cond5 = false
					}
				}
			}
			cond3 := issueDetails == nil || assignee == nil || assignee.ID == issueDetails.Assignee.ID
			cond4 := weight == nil || issueDetails == nil || *weight == *issueDetails.Weight

			sdk.LogDebug(w.logger, "debug-debug3", "cond1", cond1)
			sdk.LogDebug(w.logger, "debug-debug3", "cond2", cond2)
			sdk.LogDebug(w.logger, "debug-debug3", "cond3", cond3)
			sdk.LogDebug(w.logger, "debug-debug3", "cond4", cond4)
			sdk.LogDebug(w.logger, "debug-debug3", "cond5", cond4)

			if cond1 && cond2 && cond3 && cond4 && cond5 {
				if columnLabel.ID == api.OpenColumn {
					if issueDetails.Open {
						issues = append(issues, issueID)
					}
				} else if columnLabel.ID == api.ClosedColumn {
					if !issueDetails.Open {
						issues = append(issues, issueID)
					}
				} else {
					issues = append(issues, issueID)
				}
			}

		}
	}

	sdk.LogDebug(w.logger, "debug-debug3 returning", "issues", issues)

	return issues
}

// AddMilestoneDetails desc
func (w *WorkManager) AddMilestoneDetails(milestoneRefID int64, milestone api.Milestone) {
	w.refMilestonesDetails.Store(milestoneRefID, &milestoneDetail{
		Milestone: &milestone,
		boards:    map[string][]*api.Label{},
	})
}

// AddBoardColumnLabelToMilestone desc
func (w *WorkManager) AddBoardColumnLabelToMilestone(milestoneRefID int64, boardID string, label *api.Label) {

	sdk.LogDebug(w.logger, "debug-debug-problem", "milestoneID", milestoneRefID)
	milestoneD, _ := w.refMilestonesDetails.Load(milestoneRefID)
	milestone := milestoneD.(*milestoneDetail)
	_, ok := milestone.boards[boardID]
	if !ok {
		milestone.boards[boardID] = []*api.Label{label}
	} else {
		milestone.boards[boardID] = append(milestone.boards[boardID], label)
	}
	w.refMilestonesDetails.Store(milestoneRefID, milestone)

}

// GetSprintColumns desc
func (w *WorkManager) GetSprintColumns(milestoneRefID string) []sdk.AgileSprintColumns {
	return []sdk.AgileSprintColumns{}
}

// GetSprintIssues desc
func (w *WorkManager) GetSprintIssues(milestoneRefID string) []string {
	return []string{}
}

// GetSprintBoardsIDs desc
func (w *WorkManager) GetSprintBoardsIDs(milestoneRefID string) []string {
	return []string{}
}

// NewWorkManager desc
func NewWorkManager(logger sdk.Logger) *WorkManager {
	return &WorkManager{
		logger: logger,
	}
}
