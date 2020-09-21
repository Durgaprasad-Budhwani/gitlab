package api

import (
	"io"
	"net/url"

	"github.com/pinpt/agent.next/sdk"
)

type GitUser2 interface {
	RefID(customerID string) string
	ToModel(customerID string, integrationInstanceID string) *sdk.SourceCodeUser
}

type UserManager2 interface {
	EmitGitUser(sdk.Logger, GitUser2) error
}

type IssueManager2 interface {
	AddIssueID(issueID string, issueState string, projectID string, milestoneRefID int64, labels []int64)
	GetIssuesIDsByLabelRefIDMilestonRefID(labelRefID int64, milestoneRefID int64) []string
	GetIssuesIDsByProject(projectID string) []string
	GetIssuesIDsByMilestone(milestoneID int64) []string
	GetOpenIssuesIDsByProject(projectLabels []int64, projectID string) []string
	GetCloseIssuesIDsByProject2(projectLabels []int64, projectID string) []string
	GetOpenIssuesIDsByGroupBoardLabels(boardLabels []int64, projectIDs []string) []string
	GetClosedIssuesIDsByGroupBoardLabels(boardLabels []int64, projectIDs []string) []string
}

type SprintManager2 interface {
	AddBoardID(sprintID int64, boardID string)
	GetBoardID(sprintID int64) string
	AddColumnWithIssuesIDs(sprintID string, columnName string, issuesIDs []string)
	GetSprintColumnsIssuesIDs(sprintID string) []sdk.AgileSprintColumns
}

// QueryContext query context
type QueryContext struct {
	BaseURL string
	Logger  sdk.Logger
	Get     func(url string, params url.Values, response interface{}) (NextPage, error)
	Post    func(url string, params url.Values, data io.Reader, response interface{}) (NextPage, error)
	Delete  func(url string, params url.Values, response interface{}) (NextPage, error)

	CustomerID string
	RefType    string

	UserEmailMap          map[string]string
	IntegrationInstanceID string
	Pipe                  sdk.Pipe
	UserManager           UserManager2
	WorkManager           WorkManagerI
	SprintManager         SprintManager2
	State                 sdk.State
	Historical            bool
}

type NextPage string

type Assignee struct{}

// WorkManagerI interface to manage issues, boards, milestones, labels, columns
type WorkManagerI interface {
	// add issues with all it's details
	AddIssue(issueID string, issueState bool, projectID string, labels []*Label, milestone *Milestone, assignees *UserModel, weight *int)
	// get issues for specific column using those filters
	GetBoardColumnIssues(projectsRefIDs []string, milestone *Milestone, boardLabels []*Label, columnsLabels []BoardList, columnLabel *Label, assignee *UserModel, weight *int) []string
	// add milestone details by its ref id
	AddMilestoneDetails(milestoneRefID int64, milestone Milestone)
	// add a label to a board with milestone associated
	AddBoardColumnLabelToMilestone(milestoneRefID int64, boardID string, label *Label)
	// get sprint columns for sdk.WorkSprint
	SetSprintColumnsIssuesProjectIDs(sprint *sdk.AgileSprint)
	// get sprint board ids for sdk.WorkSprint
	GetSprintBoardsIDs(milestoneRefID string) []string
}
