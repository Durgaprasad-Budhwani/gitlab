package internal

import (
	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (g *GitlabIntegration) exportGroupProjects(group *api.Group) (projects []*sdk.WorkProject, rerr error) {

	rerr = g.exportGroupProjectsRepos(group, func(repo *sdk.SourceCodeRepo) {
		if g.IncludeRepo(group.Name, repo.Name, !repo.Active) {
			projects = append(projects, ToProject(repo))
		}

	})

	return
}

func (g *GitlabIntegration) exportUserProjects(user *api.GitlabUser) (projects []*sdk.WorkProject, rerr error) {

	rerr = g.exportUserProjectsRepos(user, func(repo *sdk.SourceCodeRepo) {
		if g.IncludeRepo(user.Name, repo.Name, !repo.Active) {
			projects = append(projects, ToProject(repo))
		}
	})

	return
}
