package api

import (
	"github.com/pinpt/agent/v4/sdk"
)

type GraphqlRequester struct {
	concurrency chan bool
	graphClient sdk.GraphQLClient
}

// NewGraphqlRequester new graphql requester
func NewGraphqlRequester(client sdk.GraphQLClient, concurrency int) GraphqlRequester {
	return GraphqlRequester{
		graphClient: client,
		concurrency: make(chan bool, concurrency),
	}
}

// Query query
func (e *GraphqlRequester) Query(query string, variables map[string]interface{}, out interface{}) error {
	e.concurrency <- true
	defer func() {
		<-e.concurrency
	}()

	return e.graphClient.Query(query, variables, out)

}
