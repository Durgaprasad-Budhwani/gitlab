package internal

import (
	"fmt"
	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
	"github.com/pinpt/gitlab/internal/common"
	"net/url"
	"strconv"
	"time"
)

func (ge *GitlabExport) exportNamespaceRepos(namespace *api.Namespace) (repos []*api.GitlabProjectInternal, rerr error) {
	rerr = ge.fetchNamespaceProjectsRepos(namespace, func(repo *api.GitlabProjectInternal) {
		if ge.IncludeRepo(namespace.ID, repo.Name, !repo.Active) {
			repos = append(repos, repo)
		}
	})
	return
}

func (ge *GitlabExport2) exportRepos(logger sdk.Logger, namespace *Namespace) ([]*api.GitlabProject, error) {

	var reposExported []*api.GitlabProject

	err :=  api.Paginate2( "", false, ge.lastExportDate, func(params url.Values, stopOnUpdatedAt time.Time) (api.NextPage, error) {
		var repos []*api.GitlabProject
		var np api.NextPage
		var err error
		if namespace.Kind == common.NamespaceGroupKind {
			np, repos, err = groupNamespaceReposPage2(logger, ge.qc, namespace, params)
			if err != nil {
				return np, err
			}
		} else {
			np, repos, err = userReposPage2(logger, ge.qc, namespace, params, stopOnUpdatedAt)
			if err != nil {
				return np, err
			}
		}
		for _, r := range repos {
			if ge.includeRepo(logger, namespace.ID, r.FullName, r.Archived) {

				reposExported = append(reposExported, r)

				repoRefID := strconv.FormatInt(r.RefID, 10)

				repo := &sdk.SourceCodeRepo{
					ID:            sdk.NewSourceCodeRepoID(ge.customerID, repoRefID, common.GitlabRefType),
					RefID:         repoRefID,
					RefType:       common.GitlabRefType,
					CustomerID:    ge.customerID,
					IntegrationInstanceID: ge.integrationInstanceID,
					Name:          r.FullName,
					URL:           r.WebURL,
					DefaultBranch: r.DefaultBranch,
					Description:   r.Description,
					UpdatedAt:     sdk.TimeToEpoch(r.UpdatedAt),
					Active:        !r.Archived,
				}

				if r.Visibility == "private" {
					repo.Visibility = sdk.SourceCodeRepoVisibilityPrivate
				} else {
					repo.Visibility = sdk.SourceCodeRepoVisibilityPublic
				}
				if r.ForkedFromProject != nil  {
					repo.Affiliation = sdk.SourceCodeRepoAffiliationThirdparty
				} else {
					repo.Affiliation = sdk.SourceCodeRepoAffiliationOrganization // Make this dynamic for user/org affiliation
				}

				if err := ge.pipe.Write(repo); err != nil{
					return "", err
				}
			} else {
				sdk.LogDebug(logger,"skipping repo","repo",r.FullName)
			}
		}
		return np, nil
	})


	return reposExported, err

}

func (ge *GitlabExport2) exportPullRequests(logger sdk.Logger, repo *int64, startPage api.NextPage) ([]*api.ApiPullRequest, error) {

	var prsExported []*api.ApiPullRequest

	err :=  api.Paginate2( startPage,false, ge.lastExportDate, func(params url.Values, stopOnUpdatedAt time.Time) (api.NextPage, error) {

		params.Set("scope", "all")
		params.Set("state", "all")

		np, prs, err := api.PullRequestPage2(logger, ge.qc, repo ,params)
		if err != nil {
			return np, fmt.Errorf("error fetching prs %s", err)
		}

		prsExported = append(prsExported, prs...)

		return np, nil
	})

	return prsExported, err

}
