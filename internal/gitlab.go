package internal

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
	pstrings "github.com/pinpt/go-common/v10/strings"
)

// GitlabIntegration is an integration for GitHub
type GitlabIntegration struct {
	logger                     sdk.Logger
	config                     sdk.Config
	manager                    sdk.Manager
	client                     sdk.GraphQLClient
	lock                       sync.Mutex
	qc                         api.QueryContext
	pipe                       sdk.Pipe
	pullrequestsFutures        []PullRequestFuture
	isssueFutures              []IssueFuture
	historical                 bool
	state                      sdk.State
	lastExportDate             time.Time
	lastExportDateGitlabFormat string
	isGitlabCom                bool
	integrationInstanceID      *string
	integrationType            string
	lastExportKey              string
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

// Mutation is called when a mutation is received on behalf of the integration
func (g *GitlabIntegration) Mutation(mutation sdk.Mutation) error {
	return nil
}

// Stop is called when the integration is shutting down for cleanup
func (g *GitlabIntegration) Stop() error {
	sdk.LogInfo(g.logger, "stopping")
	return nil
}

func (g *GitlabIntegration) setExportConfig(export sdk.Export) (err error) {

	g.pipe = export.Pipe()
	g.historical = export.Historical()
	g.state = export.State()
	g.integrationInstanceID = pstrings.Pointer(export.IntegrationInstanceID())

	logger := sdk.LogWith(g.logger, "customer_id", export.CustomerID(), "job_id", export.JobID())

	g.logger = logger

	apiURL, client, err := g.newHTTPClient(g.logger)
	if err != nil {
		return err
	}

	opts := &api.RequesterOpts{
		Concurrency: make(chan bool, 10),
		Client:      client,
		Logger:      g.logger,
	}

	requester := api.NewRequester(opts)
	g.qc.Request = requester.MakeRequest

	u, err := url.Parse(apiURL)
	if err != nil {
		return fmt.Errorf("url is not valid: %v", err)
	}
	g.isGitlabCom = u.Hostname() == "gitlab.com"

	var ok bool
	ok, g.integrationType = g.config.GetString("int_type")
	if !ok {
		return fmt.Errorf("int_type type missing")
	}

	g.lastExportKey = g.integrationType + "@last_export_date"

	if err = g.exportDate(); err != nil {
		return
	}

	g.qc.Logger = g.logger
	g.qc.RefType = GitlabRefType
	g.qc.CustomerID = export.CustomerID()

	return
}

func (g *GitlabIntegration) exportDate() (rerr error) {

	if !g.historical {
		var exportDate string
		ok, err := g.state.Get(g.lastExportKey, &exportDate)
		if err != nil {
			rerr = err
			return
		}
		if !ok {
			return
		}
		lastExportDate, err := time.Parse(time.RFC3339, exportDate)
		if err != nil {
			rerr = fmt.Errorf("error formating last export date. date %s err %s", exportDate, err)
			return
		}

		g.lastExportDate = lastExportDate.UTC()
		g.lastExportDateGitlabFormat = lastExportDate.UTC().Format(GitLabDateFormat)
	}

	sdk.LogDebug(g.logger, "last export date", "date", g.lastExportDate)

	return
}
