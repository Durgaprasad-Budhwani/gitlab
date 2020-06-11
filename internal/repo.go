package internal

import (
	"github.com/pinpt/agent.next/sdk"
)

func (g *GitlabIntegration) exportRepos(group string) (repos []*sdk.SourceCodeRepo, rerr error) {

	rerr = g.exportProjectsRepos(group, func(repo *sdk.SourceCodeRepo) {
		repos = append(repos, repo)
	})

	return
}
