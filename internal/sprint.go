package internal

import (
	"sync"

	"github.com/pinpt/agent.next/sdk"
)

// BoardManager is a manager for users
// easyjson:skip
type SprintManager struct {
	refSprintBoard        sync.Map
	refBoardColumnsIssues sync.Map
	logger                sdk.Logger
}

// AddBoardID add sprint id
func (i *SprintManager) AddBoardID(sprintID int64, boardID string) {
	i.refSprintBoard.Store(sprintID, boardID)
}

// GetBoardID get sprint id
func (i *SprintManager) GetBoardID(sprintID int64) string {
	boardID, ok := i.refSprintBoard.Load(sprintID)
	if !ok {
		return ""
	}
	return boardID.(string)
}

// AddColumnWithIssuesIDs add sprint id
func (i *SprintManager) AddColumnWithIssuesIDs(sprintID string, columnName string, issuesIDs []string) {

	sdk.LogDebug(i.logger, "adding colum with issuesIDs", "sprintID", sprintID, "issuesIDs", issuesIDs)

	agileSprintColumn, ok := i.refBoardColumnsIssues.Load(sprintID)
	if !ok {
		i.refBoardColumnsIssues.Store(sprintID, []sdk.AgileSprintColumns{{IssueIds: issuesIDs, Name: columnName}})
		return
	}

	columns := agileSprintColumn.([]sdk.AgileSprintColumns)

	columns = append(columns, sdk.AgileSprintColumns{
		Name:     columnName,
		IssueIds: issuesIDs,
	})

	i.refBoardColumnsIssues.Store(sprintID, columns)
}

func (i *SprintManager) GetSprintColumnsIssuesIDs(sprintID string) []sdk.AgileSprintColumns {
	columns, ok := i.refBoardColumnsIssues.Load(sprintID)
	if !ok {
		return []sdk.AgileSprintColumns{}
	}
	return columns.([]sdk.AgileSprintColumns)
}

// NewSprintManager returns a new instance
func NewSprintManager(logger sdk.Logger) *SprintManager {
	return &SprintManager{
		logger: logger,
	}
}
