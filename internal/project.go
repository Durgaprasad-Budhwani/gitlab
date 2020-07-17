package internal

import (
	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (ge *GitlabExport) exportGroupProjects(group *api.Group) (projects []*sdk.WorkProject, rerr error) {

	rerr = ge.exportGroupProjectsRepos(group, func(repo *sdk.SourceCodeRepo) {
		if ge.IncludeRepo(group.Name, repo.Name, !repo.Active) {
			projects = append(projects, ToProject(repo))
		}

	})

	return
}

func (ge *GitlabExport) exportUserProjects(user *api.GitlabUser) (projects []*sdk.WorkProject, rerr error) {

	rerr = ge.exportUserProjectsRepos(user, func(repo *sdk.SourceCodeRepo) {
		if ge.IncludeRepo(user.Name, repo.Name, !repo.Active) {
			projects = append(projects, ToProject(repo))
		}
	})

	return
}
