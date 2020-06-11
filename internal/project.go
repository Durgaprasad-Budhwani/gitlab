package internal

import "github.com/pinpt/agent.next/sdk"

func (g *GitlabIntegration) exportProjects(group string) (repos []*sdk.WorkProject, rerr error) {

	rerr = g.exportProjectsRepos(group, func(repo *sdk.SourceCodeRepo) {
		repos = append(repos, ToProject(repo))
	})

	return
}
