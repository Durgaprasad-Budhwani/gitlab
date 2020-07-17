package internal

import (
	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (ge *GitlabExport) exportGroupRepos(group *api.Group) (repos []*sdk.SourceCodeRepo, rerr error) {
	rerr = ge.exportGroupProjectsRepos(group, func(repo *sdk.SourceCodeRepo) {
		if ge.IncludeRepo(group.Name, repo.Name, !repo.Active) {
			repos = append(repos, repo)
		}
	})
	return
}

func (ge *GitlabExport) exportUserRepos(user *api.GitlabUser) (repos []*sdk.SourceCodeRepo, rerr error) {
	rerr = ge.exportUserProjectsRepos(user, func(repo *sdk.SourceCodeRepo) {
		if ge.IncludeRepo(user.Name, repo.Name, !repo.Active) {
			repos = append(repos, repo)
		}
	})
	return
}
