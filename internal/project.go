package internal

import (
	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (ge *GitlabExport) exportNamespaceProjects(namespace *api.Namespace) (projects []*sdk.WorkProject, rerr error) {
	rerr = ge.fetchNamespaceProjectsRepos(namespace, func(repo *sdk.SourceCodeRepo) {
		if ge.IncludeRepo(namespace.Name, repo.Name, !repo.Active) {
			projects = append(projects, ToProject(repo))
		}
	})

	return
}
