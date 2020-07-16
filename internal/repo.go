package internal

import (
	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (g *GitlabIntegration) exportGroupRepos(group *api.Group) (repos []*sdk.SourceCodeRepo, rerr error) {
	rerr = g.exportGroupProjectsRepos(group, func(repo *sdk.SourceCodeRepo) {
		if g.IncludeRepo(group.Name, repo.Name, !repo.Active) {
			repos = append(repos, repo)
		}
	})
	return
}

func (g *GitlabIntegration) exportUserRepos(user *api.GitlabUser) (repos []*sdk.SourceCodeRepo, rerr error) {
	rerr = g.exportUserProjectsRepos(user, func(repo *sdk.SourceCodeRepo) {
		if g.IncludeRepo(user.Name, repo.Name, !repo.Active) {
			repos = append(repos, repo)
		}
	})
	return
}
