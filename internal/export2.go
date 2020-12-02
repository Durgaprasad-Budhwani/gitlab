package internal

import (
	"fmt"
	"time"

	"github.com/pinpt/gitlab/internal/common"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

type includeRepo func(logger sdk.Logger, namespaceID string, name string, isArchived bool) bool

type GitlabExport2 struct {
	qc                         *api.QueryContext2
	pipe                       sdk.Pipe
	config sdk.Config
	historical                 bool
	state                      sdk.State
	lastExportDate             time.Time
	lastExportDateGitlabFormat string
	isGitlabCloud              bool
	integrationInstanceID      *string
	includeRepo                includeRepo
	baseURL                    string
	customerID string
}

// Export is called to tell the integration to run an export
func (g *GitlabIntegration) Export(export sdk.Export) error {

	logger := sdk.LogWith(export.Logger(), "job_id", export.JobID())

	qc, err := g.queryContext(logger)
	if err != nil {
		return err
	}

	ge, err := newGitlabExport(logger, qc,
		sdk.StringPointer(export.IntegrationInstanceID()),
		g.IncludeRepo(),
		export.CustomerID(),
		export.Historical(),
		export.Pipe(),
		export.State(),
		export.Config())
	if err != nil {
		return err
	}

	return ge.Export(logger)

}

func newGitlabExport(
	logger sdk.Logger,
	qc *api.QueryContext2,
	integrationInstanceID *string,
	includeRepo includeRepo,
	customerID string,
	historical bool,
	pipe sdk.Pipe,
	state sdk.State,
	config sdk.Config,
) (*GitlabExport2, error) {

	ge := &GitlabExport2{}
	ge.isGitlabCloud = qc.URL.Hostname() == "gitlab.com"
	ge.baseURL = qc.BaseURL()
	ge.pipe = pipe
	ge.state = state
	ge.config = config
	ge.historical = historical
	ge.integrationInstanceID = integrationInstanceID
	ge.customerID = customerID
	ge.qc = qc
	ge.includeRepo = includeRepo

	if !historical {
		var exportDate string
		ok, err := state.Get(common.LastExportKey, &exportDate)
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

	return ge, nil
}

func (ge *GitlabExport2) Export(logger sdk.Logger) error {

	exportStartDate := time.Now()

	sdk.LogInfo(logger, "export started", "historical", ge.historical)

	namespaces := make(chan *Namespace)
	errors := make(chan error,1)
	go func(){
		err := ge.getSelectedNamespacesIfAny(logger, namespaces)
		if err != nil {
			errors <- err
		}
	}()

	repos := make(chan *api.GitlabProject)
	go func(){
		for namespace := range namespaces {
			logger := sdk.LogWith(logger,"namespace", namespace.Name)
			sdk.LogDebug(logger, "exporting namespace")
			err := ge.exportNamespaceInitialSourceCode(logger, namespace, repos)
			if err != nil {
				sdk.LogError(logger, "error exporting namespace sourcecode", "err", err)
				errors <- err
			}
			// TODO: add WORK data type
		}
		errors <- nil
		close(repos)
	}()

	waitRepos := make(chan bool,1)
	go func(){
		for repo := range repos {
			// logger := sdk.LogWith(repo.logger)
			// TODO: write repo to pipe
			sdk.LogDebug(logger,"exporting repo","repo", repo.FullName)
		}
		waitRepos <- true
	}()

	select {
		case err := <- errors:
			if err != nil {
				return err
			}
			break
	}

	<-waitRepos

	// TODO: iterate remaining PRs of all repos
	if err := ge.state.Set(common.LastExportKey, exportStartDate.Format(time.RFC3339)); err != nil {
		return fmt.Errorf("error saving last export key in state, err = %s", err)
	}

	sdk.LogDebug(logger,"export finished")

	return nil
}

func (ge *GitlabExport2) exportNamespaceInitialSourceCode(logger sdk.Logger, namespace *Namespace, reposExported chan *api.GitlabProject) error {

	repos, err := ge.exportRepos(logger, namespace)
	if err != nil {
		return fmt.Errorf("error exporting repos %s", err)
	}

	for _, repo := range repos {
		reposExported <- repo
	}

	return nil
}

func (g *GitlabIntegration) IncludeRepo() includeRepo {
	return func(logger sdk.Logger, namespaceID string, name string, isArchived bool) bool {
		sdk.LogDebug(logger,"checking-include-logic","name", name)
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

}



func (g *GitlabIntegration) queryContext(logger sdk.Logger) (*api.QueryContext2, error) {

	c := g.config

	var authorization, restURL string

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

	qc, err := api.NewQueryContext(httpClient, restURL)
	if err != nil {
		return nil, err
	}

	return qc, nil

}


