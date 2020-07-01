package internal

import (
	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (g *GitlabIntegration) exportRepos(group *api.Group) (repos []*sdk.SourceCodeRepo, rerr error) {

	rerr = g.exportProjectsRepos(group, func(repo *sdk.SourceCodeRepo) {
		repos = append(repos, repo)
	})

	return
}
