package api

import (
	"net/url"

	"github.com/pinpt/agent.next/sdk"
)

// ServerType server type
type ServerType string

const (
	// CLOUD server type
	CLOUD ServerType = "cloud"
	// ON_PREMISE server type
	ON_PREMISE ServerType = "on-premise"
)

// QueryContext query context
type QueryContext struct {
	BaseURL string
	Logger  sdk.Logger
	Request func(url string, params url.Values, response interface{}) (PageInfo, error)

	CustomerID string
	RefType    string

	UserEmailMap map[string]string
	// IDs          ids2.Gen
	Re *RequesterOpts
}

// PageInfo page info
type PageInfo struct {
	PageSize   int
	NextPage   string
	Page       string
	TotalPages string
	Total      int
}
