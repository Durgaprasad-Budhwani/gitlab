package api

import (
	"net/url"
	"strconv"

	"github.com/pinpt/agent/v4/sdk"
)

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
	Lists     []BoardList `json:"lists"`
	Milestone *Milestone  `json:"milestone"`
	Labels    []*Label    `json:"labels"`
	Assignee  *UserModel  `json:"assignee"`
	Weight    *int        `json:"weight"`
}

func RepoBoardsPage(
	qc QueryContext,
	repo *sdk.SourceCodeRepo,
	params url.Values) (np NextPage, err error) {

	sdk.LogDebug(qc.Logger, "repo boards", "repo", repo.Name, "repo_ref_id", repo.RefID, "params", params)

	objectPath := sdk.JoinURL("projects", repo.RefID, "boards")

	initialKanbanURL := sdk.JoinURL(qc.BaseURL, "projects", repo.Name, "-", "boards", repo.RefID)

	return boardsCommonPage(qc, repo.ID, objectPath, initialKanbanURL, params, []*sdk.SourceCodeRepo{repo})
}

func GroupBoardsPage(
	qc QueryContext,
	namespace *Namespace,
	repos []*sdk.SourceCodeRepo,
	params url.Values) (np NextPage, err error) {

	sdk.LogDebug(qc.Logger, "group boards", "group", namespace.Name, "group_ref_id", namespace.ID, "params", params, "repos", repos)

	objectPath := sdk.JoinURL("groups", namespace.ID, "boards")

	initialKanbanURL := sdk.JoinURL(qc.BaseURL, "groups", namespace.Path, "-", "boards", namespace.ID)

	return boardsCommonPage(qc, namespace.ID, objectPath, initialKanbanURL, params, repos)
}

// BoardsPage boards page
func boardsCommonPage(
	qc QueryContext,
	entityID string,
	objectPath string,
	initialKanbanURL string,
	params url.Values,
	repos []*sdk.SourceCodeRepo) (np NextPage, err error) {

	var boards []Board

	np, err = qc.Get(objectPath, params, &boards)
	if err != nil {
		return
	}

	projectIDs := make([]string, 0)
	for _, repo := range repos {
		projectID := sdk.NewWorkProjectID(qc.CustomerID, repo.RefID, "gitlab")
		projectIDs = append(projectIDs, projectID)
	}

	for _, board := range boards {

		boardRefID := strconv.FormatInt(board.RefID, 10)

		sdk.LogInfo(qc.Logger, "exporting board", "name", board.Name)

		var theboard sdk.AgileBoard
		theboard.ID = sdk.NewAgileBoardID(qc.CustomerID, boardRefID, qc.RefType)
		theboard.Active = true
		theboard.CustomerID = qc.CustomerID
		theboard.RefType = qc.RefType
		theboard.RefID = boardRefID
		theboard.IntegrationInstanceID = sdk.StringPointer(qc.IntegrationInstanceID)
		theboard.Name = board.Name

		boardLists := []BoardList{{
			Label: Label{
				Name: "Open",
				ID:   OpenColumn,
			},
		}}

		for _, board := range board.Lists {
			boardLists = append(boardLists, board)
		}

		board.Lists = append(boardLists, BoardList{
			Label: Label{
				Name: "Closed",
				ID:   ClosedColumn,
			},
		})

		columns := make([]sdk.AgileBoardColumns, 0)
		for _, col := range board.Lists {
			columns = append(columns, sdk.AgileBoardColumns{
				Name: col.Label.Name,
			})
		}
		theboard.Columns = columns

		// Scrum Board
		if board.Milestone != nil {
			for _, column := range board.Lists {
				qc.WorkManager.AddBoardColumnLabelToMilestone(board.Milestone.RefID, theboard.ID, &column.Label)
			}
			theboard.Type = sdk.AgileBoardTypeScrum
		} else {
			// Kanban board
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

			for _, column := range board.Lists {
				columnIssues := qc.WorkManager.GetBoardColumnIssues(projectIDs, board.Milestone, board.Labels, board.Lists, &column.Label, board.Assignee, board.Weight)
				bc := sdk.AgileKanbanColumns{
					IssueIds: columnIssues,
					Name:     column.Label.Name,
				}
				kanban.Columns = append(kanban.Columns, bc)
				kanban.IssueIds = append(kanban.IssueIds, columnIssues...)
			}

			kanban.URL = sdk.StringPointer(initialKanbanURL)
			kanban.ID = sdk.NewAgileKanbanID(qc.CustomerID, boardRefID, qc.RefType)
			kanban.BoardID = theboard.ID

			kanban.ProjectIds = projectIDs

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
