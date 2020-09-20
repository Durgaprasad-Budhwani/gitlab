package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

type callback func(item *sdk.SourceCodeRepo)

func (ge *GitlabExport) fetchNamespaceProjectsRepos(namespace *api.Namespace, appendItem callback) (rerr error) {
	return api.Paginate(ge.logger, "", ge.lastExportDate, func(log sdk.Logger, params url.Values, stopOnUpdatedAt time.Time) (api.NextPage, error) {
		var arr []*sdk.SourceCodeRepo
		var np api.NextPage
		var err error
		if namespace.Kind == "group" {
			np, arr, err = api.GroupNamespaceReposPage(ge.qc, namespace, params, stopOnUpdatedAt)
			if err != nil {
				return np, err
			}
		} else {
			np, arr, err = api.UserReposPage(ge.qc, namespace, params, stopOnUpdatedAt)
			if err != nil {
				return np, err
			}
		}
		for _, item := range arr {
			appendItem(item)
		}
		return np, nil
	})
}

func ToProject(repo *sdk.SourceCodeRepo) *sdk.WorkProject {
	return &sdk.WorkProject{
		Active:                repo.Active,
		CustomerID:            repo.CustomerID,
		Description:           sdk.StringPointer(repo.Description),
		ID:                    repo.ID,
		Name:                  repo.Name,
		RefID:                 repo.RefID,
		RefType:               repo.RefType,
		UpdatedAt:             repo.UpdatedAt,
		URL:                   repo.URL,
		Hashcode:              repo.Hashcode,
		Identifier:            repo.Name,
		IntegrationInstanceID: repo.IntegrationInstanceID,
	}
}

func ToRepo(project *sdk.WorkProject) *sdk.SourceCodeRepo {

	repo := &sdk.SourceCodeRepo{}
	repo.Active = project.Active
	repo.CustomerID = project.CustomerID
	repo.ID = project.ID
	repo.Name = project.Name
	repo.RefID = project.RefID
	repo.RefType = project.RefType
	repo.UpdatedAt = project.UpdatedAt
	repo.URL = project.URL
	repo.Hashcode = project.Hashcode

	if project.Description == nil {
		repo.Description = ""
	} else {
		repo.Description = *project.Description
	}

	return repo
}
