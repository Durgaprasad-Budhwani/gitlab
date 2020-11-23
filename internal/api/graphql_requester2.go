package api

import (
	"fmt"
	"strings"

	"github.com/pinpt/agent/v4/sdk"
)

type GraphqlRequester2 struct {
	concurrency chan bool
	graphClient sdk.GraphQLClient
}

// NewGraphqlRequester new graphql requester
func NewGraphqlRequester2(client sdk.GraphQLClient, concurrency int) GraphqlRequester2 {
	return GraphqlRequester2{
		graphClient: client,
		concurrency: make(chan bool, concurrency),
	}
}

const retries = 3

// Query query
func (g *GraphqlRequester2) Query(logger sdk.Logger, query string, variables map[string]interface{}, out interface{}) error {
	g.concurrency <- true
	defer func() {
		<-g.concurrency
	}()

	currentRetries := 0
	for currentRetries < retries {
		err := g.graphClient.Query(query, variables, out)
		if err != nil && strings.Contains(err.Error(), "status code: 429 Too Many Requests") {
			rateLimit(logger, currentRetries)
			currentRetries++
			continue
		} else {
			return err
		}
	}

	return fmt.Errorf("too many retries")
}
