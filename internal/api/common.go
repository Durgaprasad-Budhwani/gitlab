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
}

type NextPage string
