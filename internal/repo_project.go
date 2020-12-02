package internal

import (
	"github.com/pinpt/gitlab/internal/common"
	"net/url"
	"time"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

type callback func(item *api.GitlabProjectInternal)

func (ge *GitlabExport) fetchNamespaceProjectsRepos(namespace *api.Namespace, appendItem callback) (rerr error) {
	return api.Paginate(ge.logger, "", ge.lastExportDate, func(log sdk.Logger, params url.Values, stopOnUpdatedAt time.Time) (api.NextPage, error) {
		var arr []*api.GitlabProjectInternal
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

func groupNamespaceReposPage2(logger sdk.Logger,qc *api.QueryContext2, namespace *Namespace, params url.Values) (api.NextPage, []*api.GitlabProject, error) {

	params.Set("with_shared", "false")
	params.Set("include_subgroups", "true")

	objectPath := sdk.JoinURL("groups", namespace.ID, "projects")

	return api.ReposCommonPage2(qc ,logger, params, objectPath)
}

func ToProject(repo *api.GitlabProjectInternal) *sdk.WorkProject {
	return &sdk.WorkProject{
		ID:                    sdk.NewWorkProjectID(repo.CustomerID, repo.RefID, common.GitlabRefType),
		Active:                repo.Active,
		CustomerID:            repo.CustomerID,
		Description:           sdk.StringPointer(repo.Description),
		Name:                  repo.Name,
		RefID:                 repo.RefID,
		RefType:               repo.RefType,
		UpdatedAt:             repo.UpdatedAt,
		URL:                   repo.URL,
		Hashcode:              repo.Hashcode,
		Identifier:            repo.Name,
		IntegrationInstanceID: repo.IntegrationInstanceID,
		IssueTypes: []sdk.WorkProjectIssueTypes{
			{
				RefID: api.BugIssueType,
				Name:  api.BugIssueType,
			}, {
				RefID: api.EpicIssueType,
				Name:  api.EpicIssueType,
			},
			{
				RefID: api.IncidentIssueType,
				Name:  api.IncidentIssueType,
			},
			{
				RefID: api.EnhancementIssueType,
				Name:  api.EnhancementIssueType,
			},
		},
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
