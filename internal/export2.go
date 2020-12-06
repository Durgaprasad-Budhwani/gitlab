package internal

import (
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/pinpt/gitlab/internal/common"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

type includeRepo func(logger sdk.Logger, namespaceID string, name string, isArchived bool) bool

type GitlabExport2 struct {
	qc                         *api.QueryContext2
	pipe                       sdk.Pipe
	config                     sdk.Config
	historical                 bool
	state                      sdk.State
	lastExportDate             time.Time
	lastExportDateGitlabFormat string
	isGitlabCloud              bool
	integrationInstanceID      *string
	includeRepo                includeRepo
	baseURL                    string
	customerID                 string
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
	logger       sdk.Logger
	repoRefID    *int64
	repoFullName *string
}

type remainingPrsPages struct {
	repoFullName  string
	projectID     *int64
	lastUpdatedAt string
	logger        sdk.Logger
}

type warning struct {
	msg string
}

func newWarning(msg string) warning {
	return warning{msg: msg}
}
func (n warning) Error() string {
	return n.msg
}

func (ge *GitlabExport2) Export(logger sdk.Logger) error {

	exportStartDate := time.Now()

	sdk.LogInfo(logger, "export started", "historical", ge.historical)

	var wg sync.WaitGroup

	var summaryErrors error

	namespaces := make(chan *Namespace)
	repos := make(chan *internalRepo)
	prCommits := make(chan *internalPullRequest)
	prReviews := make(chan *internalPullRequest)
	prComments := make(chan *internalPullRequest)
	remainingPrPages := make([]*remainingPrsPages, 0)

	// TODO add logic to retry writing to pipe if any
	// TODO add logic to retry failed entity page if any
	// just write to the specific channel and the logic above will do the rest
	// TODO refactor error handler per entity to not block other entities of the export
	// TODO: add WORK data type

	allNamespaces, err := api.AllNamespaces2(ge.qc, logger)
	if err != nil {
		sdk.LogError(logger, "error fetching namespaces", "err", err)
		return err
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		ge.getSelectedNamespacesIfAny(logger, allNamespaces, namespaces)
		close(namespaces)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		var wg2 sync.WaitGroup
		for pr := range prCommits { // for pr and pr commits
			wg2.Add(1)

			go func(pr *internalPullRequest) {
				defer wg2.Done()
				sdk.LogDebug(pr.logger, "exporting pr commits", "pr", pr.Title)
				err := ge.exportPullRequestCommits(pr.logger, pr)
				if err != nil {
					sdk.LogError(logger, "error exporting pr commits", "err", err)
					//errors <- err // This will be handled differently as we don't need to cancel the export if one namespace failed
				}
			}(pr)
		}
		wg2.Wait()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		var wg2 sync.WaitGroup
		for pr := range prReviews {
			wg2.Add(1)

			go func(pr *internalPullRequest) {
				defer wg2.Done()
				sdk.LogDebug(pr.logger, "exporting pr reviews", "pr", pr.Title)
				if err := ge.exportPullRequestsReviews(pr.logger, pr); err != nil {
					sdk.LogError(logger, "error exporting pr reviews", "err", err)
					// handle err
				}
			}(pr)
		}
		wg2.Wait()

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		var wg2 sync.WaitGroup
		for pr := range prComments {
			wg2.Add(1)
			go func(pr *internalPullRequest) {
				defer wg2.Done()
				sdk.LogDebug(pr.logger, "exporting pr comments", "pr", pr.Title)
				if err := ge.exportPullRequestsComments(pr.logger, pr); err != nil {
					sdk.LogError(logger, "error exporting pr comments", "err", err)
					// handle err
				}
			}(pr)
		}
		wg2.Wait()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		/// Export first 100 prs
		var wg2 sync.WaitGroup
		for repo := range repos {
			wg2.Add(1)
			go func(repo *internalRepo) {
				defer wg2.Done()

				logger := sdk.LogWith(repo.logger, "repo", repo.FullName)
				sdk.LogDebug(logger, "exporting repo")
				lastUpdatedAt, err := ge.exportPullRequestsSourceCode(logger, &repo.RefID, &repo.FullName, "1", true, "", prCommits, prReviews, prComments)
				if err != nil {
					msg := fmt.Sprintf("error exporting initial pull requests, repo %s, err %s", repo.FullName, err)
					summaryErrors = multierror.Append(summaryErrors, newWarning(msg))
				} else {
					remainingPrPages = append(remainingPrPages, &remainingPrsPages{
						projectID:     &repo.RefID,
						repoFullName:  repo.FullName,
						lastUpdatedAt: lastUpdatedAt,
						logger:        logger,
					})
				}
			}(repo)
		}
		wg2.Wait()

		var wg3 sync.WaitGroup
		for _, rpp := range remainingPrPages {
			wg3.Add(1)
			go func(rpp *remainingPrsPages) {
				defer wg3.Done()

				sdk.LogDebug(rpp.logger, "exporting remaining prs")
				_, err := ge.exportPullRequestsSourceCode(rpp.logger, rpp.projectID, &rpp.repoFullName, "2", false, rpp.lastUpdatedAt, prCommits, prReviews, prComments)
				if err != nil {
					msg := fmt.Sprintf("error exporting remaining pull requests, repo %s, err %s", rpp.repoFullName, err)
					summaryErrors = multierror.Append(summaryErrors, newWarning(msg))
				}
			}(rpp)
		}
		wg3.Wait()

		close(prCommits)
		close(prReviews)
		close(prComments)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		var wg2 sync.WaitGroup
		for namespace := range namespaces {
			//if namespace.FullPath != "josecordaz" {
				//fmt.Println("name", namespace.Name,namespace.FullPath)
			//	continue
			//}
			wg2.Add(1)
			go func(namespace *Namespace) {
				defer wg2.Done()
				logger := sdk.LogWith(logger, "namespace", namespace.Name)
				sdk.LogDebug(logger, "exporting namespace")
				if err := ge.exportRepoSourceCode(logger, namespace, repos); err != nil {
					msg := fmt.Sprintf("error exporting repos, namespace %s, err %s", namespace.Name, err)
					summaryErrors = multierror.Append(summaryErrors, newWarning(msg))
				}
			}(namespace)
		}
		wg2.Wait()
		close(repos)
	}()

	wg.Wait()

	if summaryErrors != nil {
		if merr, ok := summaryErrors.(*multierror.Error); ok {
			for _, err := range merr.Errors {
				var nsErr warning
				if errors.As(err, &nsErr) {
					sdk.LogWarn(logger, fmt.Sprintf("%s. It will be retried in the next export", err.Error()))
				} else {
					sdk.LogError(logger, err.Error())
				}
			}
		}
	}

	if err := ge.state.Set(common.LastExportKey, exportStartDate.Format(time.RFC3339)); err != nil {
		return fmt.Errorf("error saving last export key in state, err = %s", err)
	}

	sdk.LogDebug(logger, "export finished")

	return nil
}

func (ge *GitlabExport2) exportRepoSourceCode(logger sdk.Logger, namespace *Namespace, reposExported chan *internalRepo) error {

	//if namespace.Name == "premium_group2" {
	//	return fmt.Errorf("customer error for testing"+namespace.Name)
	//}

	repos, err := ge.exportRepos(logger, namespace)
	if err != nil {
		return fmt.Errorf("error exporting repos %s", err)
	}

	for _, repo := range repos {
		reposExported <- &internalRepo{
			GitlabProject: repo,
			logger:        logger,
		}
	}

	return nil
}

func (ge *GitlabExport2) exportPullRequestsSourceCode(logger sdk.Logger, repoRefID *int64, repoFullName *string, startPage api.NextPage, onlyFirstPage bool, updatedBefore string, prsCommits, prReviews, prComments chan *internalPullRequest) (string, error) {

	sdk.LogDebug(logger, "pull requests", "start_page", startPage, "only_first_page", onlyFirstPage, "updated_before", updatedBefore)

	var lastPrDate string

	err := api.Paginate2(startPage, onlyFirstPage, ge.lastExportDate, func(params url.Values, stopOnUpdatedAt time.Time) (api.NextPage, error) {

		params.Set("scope", "all")
		params.Set("state", "all")
		if updatedBefore != "" {
			params.Set("updated_before", updatedBefore)
		}

		np, prs, err := api.PullRequestPage2(logger, ge.qc, repoRefID, params)
		if err != nil {
			return np, fmt.Errorf("error fetching prs %s", err)
		}

		for _, pr := range prs {
			if lastPrDate == "" {
				lastPrDate = pr.UpdatedAt.Format(common.GitLabCreatedAtFormat)
			}
			iPr := &internalPullRequest{
				ApiPullRequest: pr,
				logger:         logger,
				repoRefID:      repoRefID,
				repoFullName:   repoFullName,
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
		sdk.LogDebug(logger, "checking-include-logic", "name", name)
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
