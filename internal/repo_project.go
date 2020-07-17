package internal

import (
	"net/url"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"

	pstrings "github.com/pinpt/go-common/v10/strings"
)

type callback func(item *sdk.SourceCodeRepo)

func (ge *GitlabExport) exportGroupProjectsRepos(group *api.Group, appendItem callback) (rerr error) {
	return api.Paginate(ge.logger, "", ge.lastExportDate, func(log sdk.Logger, params url.Values, stopOnUpdatedAt time.Time) (pi api.NextPage, err error) {
		pi, arr, err := api.GroupReposPage(ge.qc, group, params, stopOnUpdatedAt)
		if err != nil {
			return
		}
		for _, item := range arr {
			appendItem(item)
		}
		return
	})
}

func (ge *GitlabExport) exportUserProjectsRepos(user *api.GitlabUser, appendItem callback) (rerr error) {
	return api.Paginate(ge.logger, "", ge.lastExportDate, func(log sdk.Logger, params url.Values, stopOnUpdatedAt time.Time) (pi api.NextPage, err error) {
		pi, arr, err := api.UserReposPage(ge.qc, user, params, stopOnUpdatedAt)
		if err != nil {
			return
		}
		for _, item := range arr {
			appendItem(item)
		}
		return
	})
}

func ToProject(repo *sdk.SourceCodeRepo) *sdk.WorkProject {
	return &sdk.WorkProject{
		Active:      repo.Active,
		CustomerID:  repo.CustomerID,
		Description: pstrings.Pointer(repo.Description),
		ID:          repo.ID,
		Name:        repo.Name,
		RefID:       repo.RefID,
		RefType:     repo.RefType,
		UpdatedAt:   repo.UpdatedAt,
		URL:         repo.URL,
		Hashcode:    repo.Hashcode,
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
