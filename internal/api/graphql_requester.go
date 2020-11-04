package api

import (
	"strings"

	"github.com/pinpt/agent/v4/sdk"
)

type GraphqlRequester struct {
	concurrency chan bool
	graphClient sdk.GraphQLClient
	logger      sdk.Logger
}

// NewGraphqlRequester new graphql requester
func NewGraphqlRequester(client sdk.GraphQLClient, concurrency int, logger sdk.Logger) GraphqlRequester {
	return GraphqlRequester{
		graphClient: client,
		concurrency: make(chan bool, concurrency),
		logger:      logger,
	}
}

// Query query
func (e *GraphqlRequester) Query(query string, variables map[string]interface{}, out interface{}) error {
	e.concurrency <- true
	defer func() {
		<-e.concurrency
	}()

	retryCount := 0

	for {
		err := e.graphClient.Query(query, variables, out)
		if err != nil && strings.Contains(err.Error(), "status code: 429 Too Many Requests") {
			rateLimit(e.logger, retryCount)
			retryCount++
			continue
		} else {
			return err
		}
	}

}
