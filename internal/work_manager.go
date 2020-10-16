package internal

import (
	"strconv"
	"sync"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

// WorkManager work manager
type WorkManager struct {
	refProject      sync.Map
	refIssueDetails sync.Map
	logger          sdk.Logger
	state           sdk.State
}

type issueDetail struct {
	Open           bool
	Labels         map[int64]*api.Label
	Assignee       *api.UserModel
	MilestoneRefID int64
	IterationRefID string
	Weight         *int
}

type milestoneDetail struct {
	*api.Milestone
	Boards map[string][]*api.Label
}

type iterationDetail struct {
	*api.Iteration
	Boards map[string][]*api.Label
}

// AddIssue desc
func (w *WorkManager) AddIssue(issueID string, issueIID string, issueState bool, projectID string, labels []interface{}, milestone *api.Milestone, iterationRefID string, assignee *api.UserModel, weight *int) {

	var convertLabelsToMap = func() map[int64]*api.Label {

		m := make(map[int64]*api.Label)

		for _, label := range labels {
			switch label.(type) {
			case *api.Label:
				l := label.(*api.Label)
				m[l.ID] = l
			}
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
		Labels:         convertLabelsToMap(),
		Assignee:       assignee,
		MilestoneRefID: milestoneRefID,
		Weight:         weight,
		IterationRefID: iterationRefID,
	}

	projectIssues, ok := w.refProject.Load(projectID)
	if !ok {
		w.refProject.Store(projectID, map[string]*issueDetail{issueID: issueD})
	} else {
		projectIssues := projectIssues.(map[string]*issueDetail)
		projectIssues[issueID] = issueD
		w.refProject.Store(projectID, projectIssues)
	}

	w.refIssueDetails.Store(issueID, issueIID)

}

// AddIssue2 desc
func (w *WorkManager) AddIssue2(issueID string, issueIID string, issueState bool, projectID string, labels []*api.Label2, milestone *api.Milestone2, iterationRefID string, assignee *api.UserModel, weight *int) {

	var convertLabelsToMap = func() map[int64]*api.Label {

		m := make(map[int64]*api.Label)

		for _, label := range labels {
			lblID, _ := strconv.Atoi(label.ID)
			m[int64(lblID)] = &api.Label{
				ID:   int64(lblID),
				Name: label.Title,
			}
		}

		return m
	}

	var milestoneRefID int64
	if milestone != nil {
		mID := api.ExtractGraphQLID(milestone.ID)
		refID, _ := strconv.Atoi(mID)
		milestoneRefID = int64(refID)
	} else {
		milestoneRefID = 0
	}

	issueD := &issueDetail{
		Open:           issueState,
		Labels:         convertLabelsToMap(),
		Assignee:       assignee,
		MilestoneRefID: milestoneRefID,
		Weight:         weight,
		IterationRefID: iterationRefID,
	}

	projectIssues, ok := w.refProject.Load(projectID)
	if !ok {
		w.refProject.Store(projectID, map[string]*issueDetail{issueID: issueD})
	} else {
		projectIssues := projectIssues.(map[string]*issueDetail)
		projectIssues[issueID] = issueD
		w.refProject.Store(projectID, projectIssues)
	}

	w.refIssueDetails.Store(issueID, issueIID)

}

// GetBoardColumnIssues desc
func (w *WorkManager) GetBoardColumnIssues(projectsIDs []string, milestone *api.Milestone, boardLabels []*api.Label, boardLists []api.BoardList, columnLabel *api.Label, assignee *api.UserModel, weight *int) []string {

	issues := make([]string, 0)

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

	if len(projectsIDs) == 0 {
		w.refProject.Range(func(projectID, v interface{}) bool {
			projectsIDs = append(projectsIDs, projectID.(string))
			return true
		})
	}

	for _, projectID := range projectsIDs {
		projectIssues, ok := w.refProject.Load(projectID)
		if !ok {
			continue
		}

		for issueID, issueDetails := range projectIssues.(map[string]*issueDetail) {

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
			cond3 := issueDetails == nil || assignee == nil || issueDetails.Assignee == nil || assignee.ID == issueDetails.Assignee.ID
			cond4 := weight == nil || issueDetails == nil || issueDetails.Weight == nil || *weight == *issueDetails.Weight

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

	return issues
}

// SetSprintColumnsIssuesProjectIDs desc
func (w *WorkManager) SetSprintColumnsIssuesProjectIDs(sprint *sdk.AgileSprint) {

	columns := make([]sdk.AgileSprintColumns, 0)

	projectIDs := make(map[string]bool)
	unstartedIssues := make([]string, 0)
	ongoingIssues := make([]string, 0)
	completedIssues := make([]string, 0)
	{
		w.refProject.Range(func(projectID, v interface{}) bool {
			issuesMap := v.(map[string]*issueDetail)
			for issueID, issueDetail := range issuesMap {
				if issueDetail.IterationRefID == sprint.RefID {
					if issueDetail.Open == true && issueDetail.Assignee == nil {
						unstartedIssues = append(unstartedIssues, issueID)
					} else if issueDetail.Open == true && issueDetail.Assignee != nil {
						ongoingIssues = append(ongoingIssues, issueID)
					} else if !issueDetail.Open {
						completedIssues = append(completedIssues, issueID)
					}
					projectIDs[projectID.(string)] = true
				}
			}
			return true
		})
	}

	columns = []sdk.AgileSprintColumns{
		{
			Name:     "Open Issues", // ( open and unassigned )
			IssueIds: unstartedIssues,
		}, {
			Name:     "Scheduled Issues", // ( open and assigned )
			IssueIds: ongoingIssues,
		}, {
			Name:     "Closed Issues", // ( closed )
			IssueIds: completedIssues,
		},
	}

	allissues := append(append(unstartedIssues, ongoingIssues...), completedIssues...)

	sprint.Columns = columns
	sprint.IssueIds = allissues
	sprint.ProjectIds = sdk.Keys(projectIDs)

}

func convertToInt64(milestoneRefID string) int64 {
	mRefID, _ := strconv.Atoi(milestoneRefID)
	return int64(mRefID)
}

// NewWorkManager desc
func NewWorkManager(logger sdk.Logger, state sdk.State) *WorkManager {
	return &WorkManager{
		logger: sdk.LogWith(logger, "entity", "work manager"),
		state:  state,
	}
}

func (w *WorkManager) GetIssueIID(issueID string) string {

	issueIID, ok := w.refIssueDetails.Load(issueID)
	if !ok {
		return ""
	}

	return issueIID.(string)
}
