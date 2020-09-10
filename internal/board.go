package internal

import (
	"net/url"
	"sync"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (ge *GitlabExport) exportGroupBoards(namespace *api.Namespace) error {
	if namespace.Kind == "user" {
		return nil
	}

	return api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, rerr error) {
		pi, err := api.GroupBoardsPage(ge.qc, namespace, params)
		if err != nil {
			return pi, err
		}
		return
	})
}

func (ge *GitlabExport) exportReposBoards(repos []*sdk.SourceCodeRepo) error {

	for _, repo := range repos {
		err := api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, rerr error) {
			pi, err := api.RepoBoardsPage(ge.qc, repo, params)
			if err != nil {
				return pi, err
			}
			return
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// BoardManager is a manager for users
// easyjson:skip
type SprintManager struct {
	refSprintBoard        sync.Map
	refBoardColumnsIssues sync.Map
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
func (i *SprintManager) AddColumnWithIssuesIDs(sprintID string, issuesIDs []string) {
	agileSprintColumn, ok := i.refBoardColumnsIssues.Load(sprintID)
	if !ok {
		i.refBoardColumnsIssues.Store(sprintID, make([]*sdk.AgileSprintColumns, 0))
		return
	}

	columns := agileSprintColumn.([]sdk.AgileSprintColumns)

	columns = append(columns, sdk.AgileSprintColumns{
		IssueIds: issuesIDs,
	})

	i.refBoardColumnsIssues.Store(sprintID, columns)
}

func (i *SprintManager) GetSprintColumnsIssuesIDs(sprintID string) []sdk.AgileSprintColumns {
	columns, _ := i.refSprintBoard.Load(sprintID)
	return columns.([]sdk.AgileSprintColumns)
}

// NewSprintManager returns a new instance
func NewSprintManager(logger sdk.Logger) *SprintManager {
	return &SprintManager{}
}
