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
	AddIssueID(labelID int64, issueID string, projectID string, milestoneID int64)
	GetIssuesIDsByLabelID(labelID int64) []string
	GetProjectIDsByLabel(labelID int64) map[int64]bool
	GetIssuesIDsByProject(projectID string) []string
	GetIssuesIDsByMilestone(milestoneID int64) []string
	GetOpenIssuesIDsByProject(projectID string) []string
	GetCloseIssuesIDsByProject(projectID string) []string
}

type SprintManager2 interface {
	AddBoardID(sprintID int64, boardID string)
	GetBoardID(sprintID int64) string
	AddColumnWithIssuesIDs(sprintID string, issuesIDs []string)
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
	IssueManager          IssueManager2
	SprintManager         SprintManager2
}

type NextPage string
