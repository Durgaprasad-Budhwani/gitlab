package internal

import (
	"fmt"
	"net/url"
	"time"

	"github.com/pinpt/gitlab/internal/common"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

type includeRepo func(namespaceID string, name string, isArchived bool) bool

type GitlabExport2 struct {
	qc                         api.QueryContext2
	pipe                       sdk.Pipe
	historical                 bool
	state                      sdk.State
	lastExportDate             time.Time
	lastExportDateGitlabFormat string
	isGitlabCloud              bool
	integrationInstanceID      *string
	includeRepo                includeRepo
	baseURL                    string
}

// Export is called to tell the integration to run an export
func (g *GitlabIntegration) Export2(export sdk.Export) error {

	logger := sdk.LogWith(export.Logger(), "job_id", export.JobID())

	exportStartDate := time.Now()

	sdk.LogInfo(logger, "export started", "historical", export.Historical())

	config := export.Config()

	gexport, err := g.newGitlabExport(logger, export)
	if err != nil {
		return err
	}

	namespaces, err := gexport.getAllNamespaces(logger, config)
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		_ = namespace
	}

	return gexport.state.Set(common.LastExportKey, exportStartDate.Format(time.RFC3339))
}

//goland:noinspection ALL
func (g *GitlabIntegration) newGitlabExport(logger sdk.Logger, export sdk.Export) (*GitlabExport2, error) {

	restURL := "https://gitlab.com/api/v4/"
	//graphqlURL := "https://gitlab.com/api/graphql/"

	c := g.config

	var authorization string

	if c.APIKeyAuth != nil {
		if c.APIKeyAuth.URL != "" {
			restURL = sdk.JoinURL(c.APIKeyAuth.URL, "api/v4")
			//graphqlURL = sdk.JoinURL(c.APIKeyAuth.URL, "api/graphql")
		}
		authorization = "bearer " + c.APIKeyAuth.APIKey
		sdk.LogInfo(logger, "using apikey authorization", "url", restURL)
	} else if c.OAuth2Auth != nil {
		if c.OAuth2Auth.URL != "" {
			restURL = sdk.JoinURL(c.OAuth2Auth.URL, "api/v4")
			//graphqlURL = sdk.JoinURL(c.OAuth2Auth.URL, "api/graphql")
		}
		authorization = "bearer " + c.OAuth2Auth.AccessToken
		sdk.LogInfo(logger, "using oauth2 authorization")
	} else {
		return nil, fmt.Errorf("authorization not provided")
	}

	httpClient := g.manager.HTTPManager().New(restURL, map[string]string{
		"Authorization": authorization,
	})
	//graphQLClient := g.manager.GraphQLManager().New(graphqlURL, map[string]string{
	//	"Authorization": authorization,
	//})

	u, err := url.Parse(restURL)
	if err != nil {
		return nil, fmt.Errorf("url is not valid: %v", err)
	}

	ge := &GitlabExport2{}

	ge.isGitlabCloud = u.Hostname() == "gitlab.com"
	ge.baseURL = u.Scheme + "://" + u.Hostname()
	ge.pipe = export.Pipe()
	ge.historical = export.Historical()
	ge.integrationInstanceID = sdk.StringPointer(export.IntegrationInstanceID())

	r := api.NewRequester2(httpClient, common.ConcurrentAPICalls)
	ge.qc.Get = r.Get
	//ge.qc.GraphRequester = api.NewGraphqlRequester2(graphQLClient, common.ConcurrentAPICalls)
	//ge.qc.CustomerID = export.CustomerID()
	//ge.qc.IntegrationInstanceID = export.IntegrationInstanceID()
	//ge.qc.Pipe = export.Pipe()
	ge.qc.BaseURL = u.Scheme + "://" + u.Hostname()

	sdk.LogDebug(logger, "base url", "url", ge.qc.BaseURL)

	///
	ge.includeRepo = func(namespaceID string, name string, isArchived bool) bool {
		if g.config.Exclusions != nil && g.config.Exclusions.Matches(namespaceID, name) {
			// skip any repos that don't match our rule
			sdk.LogInfo(logger, "skipping repo because it matched exclusion rule", "name", name)
			return false
		}
		if g.config.Inclusions != nil && !g.config.Inclusions.Matches(namespaceID, name) {
			// skip any repos that don't match our rule
			sdk.LogInfo(logger, "skipping repo because it didn't match inclusion rule", "name", name)
			return false
		}
		if isArchived {
			sdk.LogInfo(logger, "skipping repo because it is archived", "name", name)
			return false
		}
		return true
	}

	if !export.Historical() {
		var exportDate string
		ok, err := ge.state.Get(common.LastExportKey, &exportDate)
		if err != nil {
			return nil, fmt.Errorf("error getting last export date from state %d", err)
		}
		if !ok {
			ge.historical = true
		}
		lastExportDate, err := time.Parse(time.RFC3339, exportDate)
		if err != nil {
			return nil, fmt.Errorf("error formating last export date. date %s err %s", exportDate, err)
		}

		ge.lastExportDate = lastExportDate.UTC()
		ge.lastExportDateGitlabFormat = lastExportDate.UTC().Format(common.GitLabDateTimeFormat)
	}

	sdk.LogDebug(logger, "last export date", "date", ge.lastExportDate)

	//ge.qc.Historical = ge.historical

	return ge, nil
}
