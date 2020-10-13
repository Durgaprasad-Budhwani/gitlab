package api

import (
	"io"
	"net/url"

	"github.com/pinpt/agent/v4/sdk"
)

// GitLabDateFormat gitlab layout to format dates
const GitLabDateFormat = "2006-01-02T15:04:05.000Z"

type GitUser2 interface {
	RefID(customerID string) string
	ToModel(customerID string, integrationInstanceID string) *sdk.SourceCodeUser
}

type UserManager2 interface {
	EmitGitUser(sdk.Logger, GitUser2) error
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
	State                 sdk.State
	Historical            bool
	GraphClient           sdk.GraphQLClient
}

type NextPage string

type Assignee struct{}

// WorkManagerI interface to manage issues, boards, milestones, labels, columns
type WorkManagerI interface {
	// add issues with all it's details
	AddIssue(issueID string, issueState bool, projectID string, labels []interface{}, milestone *Milestone, assignees *UserModel, weight *int)
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
	// Persist save info into state
	Persist() error
	// Restore restoer info into work manager
	Restore() error
	// Delete state
	Delete() error
}
