package api

import (
	"net/url"
	"strconv"

	"github.com/pinpt/agent.next/sdk"
)

type milestone struct {
	ID int64 `json:"id"`
}

// Label gitlab label
type Label struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// BoardList columns
type BoardList struct {
	Label Label `json:"label"`
}

// Board gitlab board
type Board struct {
	RefID     int64       `json:"id"`
	Name      string      `json:"name"`
	Project   struct{}    `json:"project"`
	Lists     []BoardList `json:"lists"`
	Milestone *milestone  `json:"milestone"`
}

func RepoBoardsPage(
	qc QueryContext,
	repo *sdk.SourceCodeRepo,
	params url.Values) (np NextPage, err error) {

	sdk.LogDebug(qc.Logger, "repo boards", "repo", repo.Name, "repo_ref_id", repo.RefID, "params", params)

	objectPath := sdk.JoinURL("projects", repo.RefID, "boards")

	initialKanbanURL := sdk.JoinURL(qc.BaseURL, "projects", repo.Name, "-", "boards", repo.RefID)

	return boardsCommonPage(qc, objectPath, initialKanbanURL, params)
}

func GroupBoardsPage(
	qc QueryContext,
	namespace *Namespace,
	params url.Values) (np NextPage, err error) {

	sdk.LogDebug(qc.Logger, "group boards", "group", namespace.Name, "group_ref_id", namespace.ID, "params", params)

	objectPath := sdk.JoinURL("groups", namespace.ID, "boards")

	initialKanbanURL := sdk.JoinURL(qc.BaseURL, "groups", namespace.Path, "-", "boards", namespace.ID)

	return boardsCommonPage(qc, objectPath, initialKanbanURL, params)
}

// BoardsPage boards page
func boardsCommonPage(
	qc QueryContext,
	objectPath string,
	initialKanbanURL string,
	params url.Values) (np NextPage, err error) {

	var boards []Board

	np, err = qc.Get(objectPath, params, &boards)
	if err != nil {
		return
	}

	for _, board := range boards {

		boardRefID := strconv.FormatInt(board.RefID, 10)

		var theboard sdk.AgileBoard
		theboard.ID = sdk.NewAgileBoardID(qc.CustomerID, boardRefID, qc.RefType)
		theboard.Active = true
		theboard.CustomerID = qc.CustomerID
		theboard.RefType = qc.RefType
		theboard.RefID = boardRefID
		theboard.IntegrationInstanceID = sdk.StringPointer(qc.IntegrationInstanceID)
		theboard.Name = board.Name

		columns := make([]sdk.AgileBoardColumns, 0)
		for _, col := range board.Lists {
			columns = append(columns, sdk.AgileBoardColumns{
				Name: col.Label.Name,
			})
		}
		theboard.Columns = columns

		theboard.BacklogIssueIds = qc.IssueManager.GetIssuesIDs(Backlog)

		// Scrum Board
		if board.Milestone != nil {
			qc.SprintManager.AddBoardID(board.Milestone.ID, sdk.NewAgileBoardID(qc.CustomerID, boardRefID, qc.RefType))
			theboard.Type = sdk.AgileBoardTypeScrum
		} else { // Kanban board
			theboard.Type = sdk.AgileBoardTypeKanban
			var kanban sdk.AgileKanban
			kanban.Active = true
			kanban.CustomerID = qc.CustomerID
			kanban.IntegrationInstanceID = sdk.StringPointer(qc.IntegrationInstanceID)
			kanban.RefID = boardRefID
			kanban.RefType = qc.RefType
			kanban.Name = board.Name
			kanban.IssueIds = make([]string, 0)
			kanban.Columns = make([]sdk.AgileKanbanColumns, 0)

			projectids := make(map[int64]bool)

			for _, column := range board.Lists {
				columnIssues := qc.IssueManager.GetIssuesIDs(column.Label.ID)
				qc.SprintManager.AddColumnWithIssuesIDs(strconv.FormatInt(board.Milestone.ID, 10), columnIssues)
				bc := sdk.AgileKanbanColumns{
					IssueIds: columnIssues,
					Name:     column.Label.Name,
				}
				kanban.Columns = append(kanban.Columns, bc)
				kanban.IssueIds = append(kanban.IssueIds, columnIssues...)

				labelProjectIDs := qc.IssueManager.GetProjectIDs(column.Label.ID)
				for projectID := range labelProjectIDs {
					projectids[projectID] = true
				}
			}
			kanban.URL = sdk.StringPointer(initialKanbanURL)
			kanban.ID = sdk.NewAgileKanbanID(qc.CustomerID, boardRefID, qc.RefType)
			kanban.BoardID = theboard.ID

			projectIDsArray := make([]string, 0)
			for projectID := range projectids {
				projectIDsArray = append(projectIDsArray, strconv.FormatInt(projectID, 10))
			}

			kanban.ProjectIds = projectIDsArray

			if err := qc.Pipe.Write(&kanban); err != nil {
				return np, err
			}
		}
		if err := qc.Pipe.Write(&theboard); err != nil {
			return np, err
		}
	}

	return
}
