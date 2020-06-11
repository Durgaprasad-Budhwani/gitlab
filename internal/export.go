package internal

import (
	"fmt"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

// Integration constants types
const (
	IntegrationSourceCodeType = "SOURCECODE"
	IntegrationWorkType       = "WORK"
)

// GitlabRefType Integraion constant type
const GitlabRefType = "gitlab"

// MaxFetchedEntitiesCount max amount items the gitlab api can retrieve
const MaxFetchedEntitiesCount = "100"

// Export is called to tell the integration to run an export
func (g *GitlabIntegration) Export(export sdk.Export) (rerr error) {

	sdk.LogDebug(g.logger, "export starting")

	ok, integrationType := export.Config().GetString("ref_type")
	if !ok {
		return fmt.Errorf("integration type missing")
	}

	g.pipe = export.Pipe()

	if rerr = g.initRequester(export); rerr != nil {
		return
	}

	g.setContextConfig(export)

	sdk.LogInfo(g.logger, "export started", "int_type", integrationType)

	groups, err := api.GroupsAll(g.qc)
	if err != nil {
		rerr = err
		return
	}

	for _, group := range groups {
		sdk.LogDebug(g.logger, "group", "name", group)
		switch integrationType {
		case IntegrationSourceCodeType:
			g.exportSourceCode(group)
		case IntegrationWorkType:
			g.exportWork(group)
		default:
			return fmt.Errorf("integration type not defined %s", integrationType)
		}
	}

	return
}

func (g *GitlabIntegration) exportSourceCode(group string) (rerr error) {

	repos, err := g.exportRepos(group)
	if err != nil {
		rerr = err
		return
	}
	for _, repo := range repos {
		if rerr = g.pipe.Write(repo); rerr != nil {
			return
		}
		if rerr = g.exportRepoPullRequests(repo); rerr != nil {
			return
		}
		if rerr = g.exportRepoUsers(repo); rerr != nil {
			return
		}
	}
	rerr = g.pipe.Flush()
	if rerr != nil {
		return
	}
	sdk.LogDebug(g.logger, "remaining pull request commits", "futures count", len(g.pullrequestsFutures))
	for _, f := range g.pullrequestsFutures {
		rerr = g.exportRemainingRepoPullRequests(f.Repo)
		if rerr != nil {
			return
		}
	}

	return
}

func (g *GitlabIntegration) exportWork(group string) (rerr error) {

	projects, err := g.exportProjects(group)
	if err != nil {
		rerr = err
		return
	}
	ProjectUsersMap := make(map[string]api.UsernameMap)
	for _, project := range projects {
		if rerr = g.pipe.Write(project); rerr != nil {
			return
		}
		users, err := g.exportProjectUsers(project)
		if err != nil {
			rerr = err
			return
		}
		ProjectUsersMap[project.RefID] = users
		if rerr = g.exportProjectIssues(project, users); rerr != nil {
			return
		}
		if rerr = g.exportProjectSprints(project); rerr != nil {
			return
		}
	}
	rerr = g.pipe.Flush()
	if rerr != nil {
		return
	}
	sdk.LogDebug(g.logger, "remaining project issues", "futures count", len(g.isssueFutures))
	for _, f := range g.isssueFutures {
		rerr = g.exportRemainingProjectIssues(f.Project, ProjectUsersMap[f.Project.RefID])
		if rerr != nil {
			return
		}
	}

	return
}
