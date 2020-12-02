package internal

import (
	"fmt"
	"sync"
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

	var wg sync.WaitGroup

	namespaces := make(chan *Namespace)
	repos := make(chan *api.GitlabProject)
	prCommits := make(chan *api.ApiPullRequest)
	prReviews := make(chan *api.ApiPullRequest)
	prComments := make(chan *api.ApiPullRequest)

	// TODO add logic to retry writting to pipe if any
	// TODO add logic to retry failed entity page if any
	// just write to the specific channel and the logic above will do the rest
	// TODO refactor error hanlder per entity to not block other entities of the export

	errors := make(chan error, 1)

	wg.Add(1)
	go func(){
		defer wg.Done()
		err := ge.getSelectedNamespacesIfAny(logger, namespaces)
		if err != nil {
			errors <- err
		}
		close(namespaces)
	}()

	// TODO: create a custom struct to save the entity and it's logger to use it afterwards with log.With

	wg.Add(1)
	go func(){
		defer wg.Done()
		for namespace := range namespaces {
			logger := sdk.LogWith(logger,"namespace", namespace.Name)
			sdk.LogDebug(logger, "exporting namespace")
			err := ge.exportRepoInitialSourceCode(logger, namespace, repos)
			if err != nil {
				sdk.LogError(logger, "error exporting namespace sourcecode", "err", err)
				errors <- err // This will be handled differently as we don't need to cancel the export if one namespace failed
			}
			// TODO: add WORK data type
		}
		close(repos)
	}()

	wg.Add(1)
	go func(){
		defer wg.Done()
		for repo := range repos {
			logger := sdk.LogWith(logger,"repo", repo.FullName)
			sdk.LogDebug(logger, "exporting repo")
			err := ge.exportPullRequestsSourceCode(logger, repo, prCommits, prReviews, prComments)
			if err != nil {
				sdk.LogError(logger, "error exporting namespace sourcecode", "err", err)
				//errors <- err // This will be handled differently as we don't need to cancel the export if one namespace failed
			}
		}
		close(prCommits)
		close(prReviews)
		close(prComments)
	}()

	wg.Add(1)
	go func(){
		defer wg.Done()
		for pr := range prCommits { // for pr and pr commits
			// logger := sdk.LogWith(repo.logger)
			sdk.LogDebug(logger,"exporting commits for pr","pr", pr.Title)
		}
	}()

	wg.Add(1)
	go func(){
		defer wg.Done()
		for pr := range prReviews {
			// logger := sdk.LogWith(repo.logger)
			sdk.LogDebug(logger,"exporting pr","pr", pr.Title)
		}
	}()

	wg.Add(1)
	go func(){
		defer wg.Done()
		for pr := range prComments {
			// logger := sdk.LogWith(repo.logger)
			sdk.LogDebug(logger,"exporting pr","pr", pr.Title)
		}
	}()

	wg.Wait()

	if len(errors) > 0 {
		// TODO: add proper handler here
	}

	// TODO: iterate remaining PRs of all repos
	if err := ge.state.Set(common.LastExportKey, exportStartDate.Format(time.RFC3339)); err != nil {
		return fmt.Errorf("error saving last export key in state, err = %s", err)
	}

	sdk.LogDebug(logger,"export finished")

	return nil
}

func (ge *GitlabExport2) exportRepoInitialSourceCode(logger sdk.Logger, namespace *Namespace, reposExported chan *api.GitlabProject) error {

	repos, err := ge.exportRepos(logger, namespace)
	if err != nil {
		return fmt.Errorf("error exporting repos %s", err)
	}

	for _, repo := range repos {
		reposExported <- repo
	}

	return nil
}

func (ge *GitlabExport2) exportPullRequestsSourceCode(logger sdk.Logger, repo *api.GitlabProject, prsExported, prReviews, prComments chan *api.ApiPullRequest) error {

	pullRequests, err := ge.exportPullRequests(logger, repo)
	if err != nil {
		return fmt.Errorf("error exporting repos %s", err)
	}

	for _, pr := range pullRequests {
		prsExported <- pr
		prReviews <- pr
		prComments <- pr
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


