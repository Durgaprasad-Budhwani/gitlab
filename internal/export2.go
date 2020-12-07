package internal

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
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
	page         string
}

type RemainingPrsPages struct {
	RepoFullName  string `json:"repoFullName"`
	ProjectID     *int64 `json:"projectID"`
	LastUpdatedAt string `json:"lastUpdatedAt"`
	logger        sdk.Logger
	Page          string `json:"page"`
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

type checkFun func(rPages map[string]interface{})
func(ge *GitlabExport2) commonRetries(logger sdk.Logger,key string, fun checkFun) error {
	rPages, err := ge.getElementsForRetries(logger, key)
	if err != nil {
		return err
	}

	fun(rPages)
	err = ge.state.Set(key, rPages)
	if err != nil {
		msg := fmt.Sprintf("error setting state key [%s]", key)
		sdk.LogDebug(logger, msg,"err", err)
		return err
	}
	return nil
}

func (ge *GitlabExport2) getElementsForRetries(logger sdk.Logger, key string) (map[string]interface{}, error) {
	var rPages map[string]interface{}
	_, err := ge.state.Get(key, &rPages)
	if err != nil {
		msg := fmt.Sprintf("error getting key [%s] from state", key)
		sdk.LogDebug(logger, msg,"err", err)
		return nil,err
	}
	if rPages == nil {
		rPages = make(map[string]interface{}, 0)
	}
	return rPages,nil
}

func (ge *GitlabExport2) appendElementForRetries(logger sdk.Logger, key string, id string, item interface{}) error {
	return ge.commonRetries(logger,key,func(rPages map[string]interface{}){
		rPages[id] = item
	})
}

func (ge *GitlabExport2) removeElementForRetries(logger sdk.Logger, key string, id string) error {
	return ge.commonRetries(logger,key, func(rPages map[string]interface{}){
		delete(rPages, id)
	})
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
	remainingPrPages := make(chan *RemainingPrsPages)

	// TODO add logic to retry writing to pipe if any
	// TODO add logic to retry failed entity page if any
	// TODO remove nextPage struct
	// just write to the specific channel and the logic above will do the rest
	// TODO refactor error handler per entity to not block other entities of the export
	// TODO: add WORK data type

	wg.Add(1)
	go func() {
		defer wg.Done()

		var wg2 sync.WaitGroup
		for pr := range prCommits { // for pr and pr commits
			wg2.Add(1)

			go func(pr *internalPullRequest) {
				defer wg2.Done()
				sdk.LogDebug(pr.logger, "exporting pr commits", "pr", pr.Title)
				lastPage, err := ge.exportPullRequestCommits(pr.logger, pr)
				if err != nil {
					msg := fmt.Sprintf("error exporting pr commits, pr %s, err %s", pr.Title, err)
					summaryErrors = multierror.Append(summaryErrors, newWarning(msg))
					pr.page = lastPage
					// save the prComment object with the proper page
				} else if lastPage == "" {
					// save date for prComment so it is used in the next incremental
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
				} else {
					// review finished correctly
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
				lastPage, err := ge.exportPullRequestsComments(pr.logger, pr)
				if err != nil {
					msg := fmt.Sprintf("error exporting pr comments, pr %s, err %s", pr.Title, err)
					summaryErrors = multierror.Append(summaryErrors, newWarning(msg))
					pr.page = lastPage
					// append the error
					// save the prComment object with the proper page
				} else if lastPage == "" {
					// save date for prcomment so it is used in the next incremental
				}
			}(pr)
		}
		wg2.Wait()
	}()

	wg.Add(1)
	go func(){
		defer wg.Done()

		var wg2 sync.WaitGroup
		for rpp := range remainingPrPages {
			wg2.Add(1)
			go func(rpp *RemainingPrsPages) {
				defer wg2.Done()

				sdk.LogDebug(rpp.logger, "exporting remaining prs")
				_, lastPage, err := ge.exportPullRequestsSourceCode(
					rpp.logger,
					rpp.ProjectID,
					&rpp.RepoFullName,
					api.NextPage(rpp.Page),
					false,
					rpp.LastUpdatedAt,
					prCommits,
					prReviews,
					prComments,
				)
				if err != nil {

					msg := fmt.Sprintf("error exporting remaining pull requests, repo %s, err %s", rpp.RepoFullName, err)
					summaryErrors = multierror.Append(summaryErrors, newWarning(msg))

					if lastPage != "" {
						rpp.Page = lastPage
					}

					if err := ge.appendElementForRetries(rpp.logger, common.RemainingPrsPagesKey,strconv.FormatInt(*rpp.ProjectID,10), *rpp); err != nil {
						return
					}
				} else if lastPage == "" {
					if err := ge.removeElementForRetries(rpp.logger, common.RemainingPrsPagesKey,strconv.FormatInt(*rpp.ProjectID,10)); err != nil {
						return
					}
					// TODO: save date of this repo in state so it is used in the next incremental
				}
			}(rpp)
		}
		wg2.Wait()

		close(prCommits)
		close(prReviews)
		close(prComments)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		/// Export first 100 prs
		var wg2 sync.WaitGroup
		remainingPrPagesArr := make([]*RemainingPrsPages,0)
		for repo := range repos {
			wg2.Add(1)
			go func(repo *internalRepo) {
				defer wg2.Done()

				logger := sdk.LogWith(repo.logger, "repo", repo.FullName)
				sdk.LogDebug(logger, "exporting repo")
				lastUpdatedAt, lastPage, err := ge.exportPullRequestsSourceCode(logger, &repo.RefID, &repo.FullName, "1", true, "", prCommits, prReviews, prComments)
				if err != nil {
					msg := fmt.Sprintf("error exporting initial pull requests, repo %s, err %s", repo.FullName, err)
					summaryErrors = multierror.Append(summaryErrors, newWarning(msg))
				} else if lastPage != "" {
					remainingPrPagesArr = append(remainingPrPagesArr, &RemainingPrsPages{
						ProjectID:     &repo.RefID,
						RepoFullName:  repo.FullName,
						LastUpdatedAt: lastUpdatedAt,
						logger:        logger,
						Page:          "2",
					})
				}
			}(repo)
		}
		wg2.Wait()

		go func(){
			for _, rpp := range remainingPrPagesArr {
				remainingPrPages <- rpp
			}
			close(remainingPrPages)
		}()


	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		var wg2 sync.WaitGroup
		for namespace := range namespaces {
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

	// Read pending remainingPrs
	rPages, err := ge.getElementsForRetries(logger, common.RemainingPrsPagesKey)
	if err != nil {
		return err
	}
	if len(rPages) > 0 {
		sdk.LogDebug(logger,"processing pending remaining pr pages","count",len(rPages))
	}
	for _, remainingPrsPage := range rPages {
		// TODO: recover the right logger
		rPP := remainingPrsPage.(RemainingPrsPages)
		rPP.logger = logger
		remainingPrPages <- &rPP
	}
	// Read pending prComments
	// Read pending prCommits
	// Read pending prReviews

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

type PageError struct {
	msg  string
	page string
}

func (p PageError) Error() string {
	return p.msg
}

func NewPageError(msg string, page string) PageError {
	return PageError{
		msg:  msg,
		page: page,
	}
}

func (ge *GitlabExport2) exportPullRequestsSourceCode(logger sdk.Logger, repoRefID *int64, repoFullName *string, startPage api.NextPage, onlyFirstPage bool, updatedBefore string, prsCommits, prReviews, prComments chan *internalPullRequest) (string, string, error) {

	sdk.LogDebug(logger, "pull requests", "start_page", startPage, "only_first_page", onlyFirstPage, "updated_before", updatedBefore)

	var lastPrDate string
	var lastNextPage api.NextPage

	err := api.Paginate2(startPage, onlyFirstPage, ge.lastExportDate, func(params url.Values, stopOnUpdatedAt time.Time) (api.NextPage, error) {

		params.Set("scope", "all")
		params.Set("state", "all")
		if updatedBefore != "" {
			params.Set("updated_before", updatedBefore)
		}

		np, prs, err := api.PullRequestPage2(logger, ge.qc, repoRefID, params)
		lastNextPage = np
		if err != nil {
			msg := fmt.Sprintf("error fetching prs %s", err)
			return np, NewPageError(msg, params.Get("page"))
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
				page: "1",
			}
			prsCommits <- iPr
			prReviews <- iPr
			prComments <- iPr
		}

		return np, nil
	})

	return lastPrDate, string(lastNextPage), err
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
