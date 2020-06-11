package internal

import (
	"fmt"
	"sync"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

// GitlabIntegration is an integration for GitHub
type GitlabIntegration struct {
	logger              sdk.Logger
	config              sdk.Config
	manager             sdk.Manager
	client              sdk.GraphQLClient
	lock                sync.Mutex
	qc                  api.QueryContext
	pipe                sdk.Pipe
	pullrequestsFutures []PullRequestFuture
	isssueFutures       []IssueFuture
}

var _ sdk.Integration = (*GitlabIntegration)(nil)

// Start is called when the integration is starting up
func (g *GitlabIntegration) Start(logger sdk.Logger, config sdk.Config, manager sdk.Manager) error {
	g.logger = sdk.LogWith(logger, "pkg", GitlabRefType)
	g.config = config
	g.manager = manager
	sdk.LogInfo(g.logger, "starting")
	return nil
}

// Enroll is called when a new integration instance is added
func (g *GitlabIntegration) Enroll(instance sdk.Instance) error {
	// FIXME: add the web hook for this integration
	return nil
}

// Dismiss is called when an existing integration instance is removed
func (g *GitlabIntegration) Dismiss(instance sdk.Instance) error {
	// FIXME: remove integration
	return nil
}

// WebHook is called when a webhook is received on behalf of the integration
func (g *GitlabIntegration) WebHook(webhook sdk.WebHook) error {
	return nil
}

// Stop is called when the integration is shutting down for cleanup
func (g *GitlabIntegration) Stop() error {
	sdk.LogInfo(g.logger, "stopping")
	return nil
}

// Start is called when the integration is starting up
func initRequesterConfig(export sdk.Export) (err error, opts api.RequesterOpts) {

	config := export.Config()
	ok, url := config.GetString("url")
	if !ok {
		url = "https://gitlab.com/api/v4/"
	}

	ok, apikey := config.GetString("api_key")
	if !ok {
		err = fmt.Errorf("required api_key not found")
		return
	}

	opts = api.RequesterOpts{
		APIURL:             url,
		APIKey:             apikey,
		InsecureSkipVerify: true,
	}

	return

}

func (g *GitlabIntegration) initRequester(export sdk.Export) (err error) {

	err, config := initRequesterConfig(export)
	if err != nil {
		return err
	}

	config.Logger = g.logger

	requester := api.NewRequester(config)
	g.qc.Request = requester.MakeRequest

	return
}

func (g *GitlabIntegration) setContextConfig(export sdk.Export) {
	g.qc.Logger = g.logger
	g.qc.RefType = GitlabRefType
	g.qc.CustomerID = export.CustomerID()
}
