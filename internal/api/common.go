package api

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/pinpt/agent/v4/sdk"
)



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

// QueryContext2 query context
type QueryContext2 struct {
	BaseURL string
	Get     func(logger sdk.Logger,url string, params url.Values, response interface{}) (NextPage, error)
	//Post    func(url string, params url.Values, data io.Reader, response interface{}) (NextPage, error)
	//Delete  func(url string, params url.Values, response interface{}) (NextPage, error)
	//Put     func(url string, params url.Values, data io.Reader, response interface{}) (NextPage, error)

	//CustomerID string
	//IntegrationInstanceID string
	//UserEmailMap          map[string]string
	//Pipe                  sdk.Pipe
	//UserManager           UserManager2
	//WorkManager           WorkManagerI
	//State                 sdk.State
	//Historical            bool
	//GraphRequester        GraphqlRequester2
}

type NextPage string

type Assignee struct{}

// WorkManagerI interface to manage issues, boards, milestones, labels, columns
type WorkManagerI interface {
	// add issues with all it's details
	AddIssue(issueID string, issueState bool, projectID string, labels []interface{}, milestone *Milestone, iterationRefID string, assignee *UserModel, weight *int, issue *IssueStateInfo)
	// add issue with full labels
	AddIssue2(issueID string, issueState bool, projectID string, labels []*Label2, milestone *Milestone2, iterationRefID string, assignee *UserModel, weight *int, issue *IssueStateInfo)
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
	// get issues details by id
	GetIssueDetails(issueID string) *IssueStateInfo
	// add project details to state
	AddProjectDetails(projectID string, project *ProjectStateInfo)
	// get project details to state
	GetProjectDetails(projectID string) *ProjectStateInfo
}

type IssueStateInfo struct {
	IID          string
	ProjectRefID string
}

type ProjectStateInfo struct {
	ProjectPath string
	GroupPath   string
}

func ExtractGraphQLID(id string) string {
	ind := strings.LastIndex(id, "/")

	return id[ind+1:]
}

func checkPermissionsIssue(logger sdk.Logger, err error, msg string) bool {
	if strings.Contains(err.Error(), "The resource that you are attempting to access does not exist or you don't have permission to perform this action") {
		sdk.LogWarn(logger, msg)
		return true
	}
	return false
}

func NewHTTPClient(logger sdk.Logger, config sdk.Config, manager sdk.Manager) (url string, cl sdk.HTTPClient, cl2 sdk.GraphQLClient, err error) {

	url = "https://gitlab.com/api/v4/"
	graphqlURL := "https://gitlab.com/api/graphql/"

	if config.APIKeyAuth != nil {
		apikey := config.APIKeyAuth.APIKey
		if config.APIKeyAuth.URL != "" {
			url = sdk.JoinURL(config.APIKeyAuth.URL, "api/v4")
			graphqlURL = sdk.JoinURL(config.APIKeyAuth.URL, "api/graphql/")
		}
		headers := map[string]string{
			"Authorization": "bearer " + apikey,
		}
		cl = manager.HTTPManager().New(url, headers)
		cl2 = manager.GraphQLManager().New(graphqlURL, headers)
		sdk.LogInfo(logger, "using apikey authorization", "apikey", apikey, "url", url)
	} else if config.OAuth2Auth != nil {
		authToken := config.OAuth2Auth.AccessToken
		if config.OAuth2Auth.URL != "" {
			url = sdk.JoinURL(config.OAuth2Auth.URL, "api/v4")
			graphqlURL = sdk.JoinURL(config.OAuth2Auth.URL, "api/graphql/")
		}
		headers := map[string]string{
			"Authorization": "bearer " + authToken,
		}
		cl = manager.HTTPManager().New(url, headers)
		cl2 = manager.GraphQLManager().New(graphqlURL, headers)
		sdk.LogInfo(logger, "using oauth2 authorization")
	} else if config.BasicAuth != nil {
		// TODO: check if this type is supported by gitlab
		if config.BasicAuth.URL != "" {
			url = sdk.JoinURL(config.BasicAuth.URL, "api/v4")
			graphqlURL = sdk.JoinURL(config.BasicAuth.URL, "api/graphql/")
		}
		headers := map[string]string{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(config.BasicAuth.Username+":"+config.BasicAuth.Password)),
		}
		cl = manager.HTTPManager().New(url, headers)
		cl2 = manager.GraphQLManager().New(graphqlURL, headers)
		sdk.LogInfo(logger, "using basic authorization", "username", config.BasicAuth.Username)
	} else {
		err = fmt.Errorf("supported authorization not provided. support for: apikey, oauth2, basic")
		return
	}
	return
}
