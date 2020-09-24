package internal

import (
	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (ge *GitlabExport) exportNamespaceRepos(namespace *api.Namespace) (repos []*sdk.SourceCodeRepo, rerr error) {
	rerr = ge.fetchNamespaceProjectsRepos(namespace, func(repo *sdk.SourceCodeRepo) {
		if ge.IncludeRepo(namespace.ID, repo.Name, !repo.Active) {
			repos = append(repos, repo)
		}
	})
	return
}
