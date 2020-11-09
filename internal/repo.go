package internal

import (
	"github.com/pinpt/gitlab/internal/api"
)

func (ge *GitlabExport) exportNamespaceRepos(namespace *api.Namespace) (repos []*api.GitlabProjectInternal, rerr error) {
	rerr = ge.fetchNamespaceProjectsRepos(namespace, func(repo *api.GitlabProjectInternal) {
		if ge.IncludeRepo(namespace.ID, repo.Name, !repo.Active) {
			repos = append(repos, repo)
		}
	})
	return
}
