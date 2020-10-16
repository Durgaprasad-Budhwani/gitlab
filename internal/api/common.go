package api

import (
	"io"
	"net/url"
	"strings"

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
	Put     func(url string, params url.Values, data io.Reader, response interface{}) (NextPage, error)

	CustomerID string
	RefType    string

	UserEmailMap          map[string]string
	IntegrationInstanceID string
	Pipe                  sdk.Pipe
	UserManager           UserManager2
	WorkManager           WorkManagerI
	State                 sdk.State
	Historical            bool
	GraphRequester        GraphqlRequester
}

type NextPage string

type Assignee struct{}

// WorkManagerI interface to manage issues, boards, milestones, labels, columns
type WorkManagerI interface {
	// add issues with all it's details
	AddIssue(issueID string, iid string, issueState bool, projectID string, labels []interface{}, milestone *Milestone, iterationRefID string, assignee *UserModel, weight *int)
	// add issue with full labels
	AddIssue2(issueID string, iid string, issueState bool, projectID string, labels []*Label2, milestone *Milestone2, iterationRefID string, assignee *UserModel, weight *int)
	// get issues for specific column using those filters
	GetBoardColumnIssues(projectsRefIDs []string, milestone *Milestone, boardLabels []*Label, columnsLabels []BoardList, columnLabel *Label, assignee *UserModel, weight *int) []string
	// get sprint columns for sdk.WorkSprint
	SetSprintColumnsIssuesProjectIDs(sprint *sdk.AgileSprint)
	// Persist save info into state
	Persist() error
	// Restore restoer info into work manager
	Restore() error
	// Delete state
	Delete() error
	// get issues for specific column using those filters
	GetIssueIID(id string) string
}

func ExtractGraphQLID(id string) string {
	ind := strings.LastIndex(id, "/")

	return id[ind+1:]
}
