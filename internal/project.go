package internal

import (
	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (g *GitlabIntegration) exportProjects(group *api.Group) (repos []*sdk.WorkProject, rerr error) {

	rerr = g.exportProjectsRepos(group, func(repo *sdk.SourceCodeRepo) {
		repos = append(repos, ToProject(repo))
	})

	return
}
