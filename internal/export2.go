package internal

import (
	"fmt"
	"net/url"
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

type internalRepo struct {
	*api.GitlabProject
	logger sdk.Logger
}

type internalPullRequest struct {
	*api.ApiPullRequest
	logger sdk.Logger
	repoRefID *int64
	repoFullName *string
}

type remainingPrsPages struct {
	repoFullName string
	projectID *int64
	lastUpdatedAt string
	logger sdk.Logger
}

func (ge *GitlabExport2) Export(logger sdk.Logger) error {

	exportStartDate := time.Now()

	sdk.LogInfo(logger, "export started", "historical", ge.historical)

	var wg sync.WaitGroup

	namespaces := make(chan *Namespace)
	repos := make(chan *internalRepo)
	prCommits := make(chan *internalPullRequest)
	prReviews := make(chan *internalPullRequest)
	prComments := make(chan *internalPullRequest)
	remainingPrPages := make([]*remainingPrsPages,0)

	// TODO add logic to retry writting to pipe if any
	// TODO add logic to retry failed entity page if any
	// just write to the specific channel and the logic above will do the rest
	// TODO refactor error hanlder per entity to not block other entities of the export
	// TODO: add WORK data type

	errors := make(chan error, 1)

	wg.Add(1)
	go func(){
		defer wg.Done()
		err := ge.getSelectedNamespacesIfAny(logger, namespaces)
		if err != nil {
			sdk.LogError(logger, "error exporting namespaces", "err", err)
			errors <- err
		}
		close(namespaces)
	}()

	// TODO: create a custom struct to save the entity and it's logger to use it afterwards with log.With

	wg.Add(1)
	go func(){
		defer wg.Done()

		totalRepos := 0
		for namespace := range namespaces {
			logger := sdk.LogWith(logger,"namespace", namespace.Name)
			sdk.LogDebug(logger, "exporting namespace")
			reposCount, err := ge.exportRepoSourceCode(logger, namespace, repos)
			if err != nil {
				sdk.LogError(logger, "error exporting repos", "err", err)
				errors <- err // This will be handled differently as we don't need to cancel the export if one namespace failed
			}
			totalRepos += reposCount
		}
		close(repos)
	}()

	wg.Add(1)
	go func(){
		defer wg.Done()
		/// Export first 100 prs
		for repo := range repos {
			logger := sdk.LogWith(repo.logger,"repo", repo.FullName)
			sdk.LogDebug(logger, "exporting repo")
			lastUpdatedAt, err := ge.exportPullRequestsSourceCode(logger, &repo.RefID, &repo.FullName,"1",true,"", prCommits, prReviews, prComments)
			if err != nil {
				sdk.LogError(logger, "error exporting initial pull requests", "err", err)
				//errors <- err // This will be handled differently as we don't need to cancel the export if one namespace failed
			}
			remainingPrPages = append(remainingPrPages, &remainingPrsPages{
				projectID: &repo.RefID,
				repoFullName: repo.FullName,
				lastUpdatedAt: lastUpdatedAt,
				logger: logger,
			})
		}

		/// Export remaining pr pages
		for _, rpp := range remainingPrPages {
			sdk.LogDebug(rpp.logger,"exporting remaining prs")
			_, err := ge.exportPullRequestsSourceCode(rpp.logger, rpp.projectID, &rpp.repoFullName,"2",false, rpp.lastUpdatedAt, prCommits, prReviews, prComments)
			if err != nil {
				sdk.LogError(logger, "error exporting remaining pull requests", "err", err)
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
			sdk.LogDebug(pr.logger,"exporting pr commits","pr", pr.Title)
			err := ge.exportPullRequestCommits(pr.logger, pr)
			if err != nil {
				sdk.LogError(logger, "error exporting pr commits", "err", err)
				//errors <- err // This will be handled differently as we don't need to cancel the export if one namespace failed
			}
		}
	}()

	wg.Add(1)
	go func(){
		defer wg.Done()
		for pr := range prReviews {
			sdk.LogDebug(pr.logger,"exporting pr reviews","pr", pr.Title)
			if err := ge.exportPullRequestsReviews(pr.logger, pr); err != nil {
				sdk.LogError(logger, "error exporting pr reviews", "err", err)
				// handle err
			}
		}
	}()

	wg.Add(1)
	go func(){
		defer wg.Done()
		for pr := range prComments {
			// logger := sdk.LogWith(repo.logger)
			sdk.LogDebug(pr.logger,"exporting pr comments","pr", pr.Title)
			if err := ge.exportPullRequestsComments(pr.logger, pr); err != nil {
				sdk.LogError(logger, "error exporting pr comments", "err", err)
				// handle err
			}
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

func (ge *GitlabExport2) exportRepoSourceCode(logger sdk.Logger, namespace *Namespace, reposExported chan *internalRepo) (int, error) {

	repos, err := ge.exportRepos(logger, namespace)
	if err != nil {
		return 0, fmt.Errorf("error exporting repos %s", err)
	}

	for _, repo := range repos {
		reposExported <- &internalRepo{
			GitlabProject: repo,
			logger: logger,
		}
	}

	return len(repos), nil
}

func (ge *GitlabExport2) exportPullRequestsSourceCode(logger sdk.Logger, repoRefID *int64,repoFullName *string, startPage api.NextPage,onlyFirstPage bool,  updatedBefore string,  prsCommits, prReviews, prComments chan *internalPullRequest) (string, error) {

	sdk.LogDebug(logger, "pull requests","start_page",startPage, "only_first_page", onlyFirstPage, "updated_before", updatedBefore)

	var lastPrDate string

	err := api.Paginate2( startPage, onlyFirstPage, ge.lastExportDate, func(params url.Values, stopOnUpdatedAt time.Time) (api.NextPage, error) {

		params.Set("scope", "all")
		params.Set("state", "all")
		if updatedBefore != "" {
			params.Set("updated_before", updatedBefore)
		}

		np, prs, err := api.PullRequestPage2(logger, ge.qc, repoRefID ,params)
		if err != nil {
			return np, fmt.Errorf("error fetching prs %s", err)
		}

		for _, pr := range prs {
			if lastPrDate == "" {
				lastPrDate = pr.UpdatedAt.Format(common.GitLabCreatedAtFormat)
			}
			iPr := &internalPullRequest{
				ApiPullRequest: pr,
				logger: logger,
				repoRefID: repoRefID,
				repoFullName: repoFullName,
			}
			prsCommits <- iPr
			prReviews <- iPr
			prComments <- iPr
		}

		return np, nil
	})

	return lastPrDate, err
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


