package internal

import (
	"fmt"
	"time"

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

// GitLabDateFormat gitlab layout to format dates
const GitLabDateFormat = "2006-01-02T15:04:05.000Z"

// Export is called to tell the integration to run an export
func (g *GitlabIntegration) Export(export sdk.Export) (rerr error) {

	sdk.LogDebug(g.logger, "export starting")

	ok, integrationType := export.Config().GetString("int_type")
	if !ok {
		return fmt.Errorf("integration type missing")
	}

	lastExportKey := integrationType + "@last_export_date"

	// TODO: Add suport for multiple test repos
	_, repo := export.Config().GetString("repo")

	if rerr = g.initRequester(export); rerr != nil {
		return
	}

	g.setExportConfig(export)

	if rerr = g.exportDate(export, lastExportKey); rerr != nil {
		return
	}

	exportStartDate := time.Now()

	sdk.LogInfo(g.logger, "export started", "int_type", integrationType)

	if repo != "" {
		switch integrationType {
		case IntegrationSourceCodeType:
			g.exportIndividualRepo(repo)
		case IntegrationWorkType:
			g.exportIndividualProject(repo)
		default:
			return fmt.Errorf("integration type not defined %s", integrationType)
		}

		rerr = g.state.Set(lastExportKey, exportStartDate.Format(time.RFC3339))
		return
	}

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

	rerr = g.state.Set(lastExportKey, exportStartDate.Format(time.RFC3339))

	return
}

func (g *GitlabIntegration) exportSourceCode(group string) (rerr error) {

	repos, err := g.exportRepos(group)
	if err != nil {
		rerr = err
		return
	}
	for _, repo := range repos {
		err := g.exportRepoAndWrite(repo)
		if err != nil {
			sdk.LogError(g.logger, "error exporting repo", "repo", repo.Name, "repo_refid", repo.RefID, "err", err)
		}
	}
	rerr = g.pipe.Flush()
	if rerr != nil {
		return
	}
	rerr = g.exportPullRequestsFutures()
	if rerr != nil {
		return
	}

	return
}

func (g *GitlabIntegration) exportRepoAndWrite(repo *sdk.SourceCodeRepo) (rerr error) {
	if rerr = g.pipe.Write(repo); rerr != nil {
		return
	}
	g.exportRepoPullRequests(repo)
	if rerr = g.exportRepoUsers(repo); rerr != nil {
		return
	}
	return
}

func (g *GitlabIntegration) exportProjectAndWrite(project *sdk.WorkProject, projectUsersMap map[string]api.UsernameMap) (rerr error) {
	if rerr = g.pipe.Write(project); rerr != nil {
		return
	}
	users, err := g.exportProjectUsers(project)
	if err != nil {
		rerr = err
		return
	}
	projectUsersMap[project.RefID] = users
	g.exportProjectIssues(project, users)
	if rerr = g.exportProjectSprints(project); rerr != nil {
		return
	}
	return
}

func (g *GitlabIntegration) exportPullRequestsFutures() (rerr error) {

	sdk.LogDebug(g.logger, "remaining pull requests", "futures count", len(g.pullrequestsFutures))

	for _, f := range g.pullrequestsFutures {
		g.exportRemainingRepoPullRequests(f.Repo)
	}

	return
}

func (g *GitlabIntegration) exportIssuesFutures(projectUsersMap map[string]api.UsernameMap) (rerr error) {

	sdk.LogDebug(g.logger, "remaining issues", "futures count", len(g.isssueFutures))

	for _, f := range g.isssueFutures {
		g.exportRemainingProjectIssues(f.Project, projectUsersMap[f.Project.RefID])
	}

	return
}

func (g *GitlabIntegration) exportWork(group string) (rerr error) {

	projects, err := g.exportProjects(group)
	if err != nil {
		rerr = err
		return
	}
	projectUsersMap := make(map[string]api.UsernameMap)
	for _, project := range projects {
		err := g.exportProjectAndWrite(project, projectUsersMap)
		if err != nil {
			sdk.LogError(g.logger, "error exporting project", "project", project.Name, "project_refid", project.RefID, "err", err)
		}
	}
	rerr = g.pipe.Flush()
	if rerr != nil {
		return
	}
	sdk.LogDebug(g.logger, "remaining project issues", "futures count", len(g.isssueFutures))
	rerr = g.exportIssuesFutures(projectUsersMap)
	if rerr != nil {
		return
	}

	return
}

func (g *GitlabIntegration) exportIndividualRepo(r string) (rerr error) {

	repo, err := api.Repo(g.qc, r)
	if err != nil {
		return err
	}
	if err := g.exportRepoAndWrite(repo); err != nil {
		return err
	}
	if err := g.exportPullRequestsFutures(); err != nil {
		return err
	}

	return
}

func (g *GitlabIntegration) exportIndividualProject(p string) (rerr error) {

	project, err := api.Repo(g.qc, p)
	if err != nil {
		return err
	}
	projectUsersMap := make(map[string]api.UsernameMap)
	if err := g.exportProjectAndWrite(ToProject(project), projectUsersMap); err != nil {
		return err
	}
	if err := g.exportIssuesFutures(projectUsersMap); err != nil {
		return err
	}

	return
}
