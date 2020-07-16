package internal

import (
	"encoding/base64"
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

	if rerr = g.setExportConfig(export); rerr != nil {
		return
	}

	sdk.LogInfo(g.logger, "export started", "historical", g.historical, "int_type", g.integrationType)

	exportStartDate := time.Now()

	orgs := make([]*api.Group, 0)
	users := make([]*api.GitlabUser, 0)
	if g.config.Accounts == nil {
		groups, err := api.GroupsAll(g.qc)
		if err != nil {
			rerr = err
			return
		}
		orgs = append(orgs, groups...)

		user, err := api.LoginUser(g.qc)
		if err != nil {
			return err
		}
		users = append(users, user)
	} else {
		for _, acct := range *g.config.Accounts {
			if acct.Type == sdk.ConfigAccountTypeOrg {
				orgs = append(orgs, &api.Group{ID: acct.ID})
			} else {
				users = append(users, &api.GitlabUser{ID: acct.ID})
			}
		}
	}

	for _, group := range orgs {
		sdk.LogDebug(g.logger, "group", "name", group.Name)
		switch g.integrationType {
		case IntegrationSourceCodeType:
			err := g.exportGroupSourceCode(group)
			if err != nil {
				sdk.LogWarn(g.logger, "error exporting sourcecode group", "group_id", group.ID, "group_name", group.Name, "err", err)
			}
		case IntegrationWorkType:
			err := g.exportGroupWork(group)
			if err != nil {
				sdk.LogWarn(g.logger, "error exporting sourcecode group", "group_id", group.ID, "group_name", group.Name, "err", err)
			}
		default:
			return fmt.Errorf("integration type not defined %s", g.integrationType)
		}
	}

	for _, user := range users {
		sdk.LogDebug(g.logger, "user", "name", user.Name)
		switch g.integrationType {
		case IntegrationSourceCodeType:
			if err := g.exportUserSourceCode(user); err != nil {
				sdk.LogWarn(g.logger, "error exporting work user", "user_id", user.ID, "user_name", user.Name, "err", err)
			}
		case IntegrationWorkType:
			if err := g.exportUserWork(user); err != nil {
				sdk.LogWarn(g.logger, "error exporting work user", "user_id", user.ID, "user_name", user.Name, "err", err)
			}
		default:
			return fmt.Errorf("integration type not defined %s", g.integrationType)
		}
	}

	rerr = g.state.Set(g.lastExportKey, exportStartDate.Format(time.RFC3339))

	return
}

func (g *GitlabIntegration) exportGroupSourceCode(group *api.Group) (rerr error) {

	if !g.isGitlabCom {
		if err := g.exportEnterpriseUsers(); err != nil {
			rerr = err
			return
		}
	}

	repos, err := g.exportGroupRepos(group)
	if err != nil {
		rerr = err
		return
	}

	return g.exportCommonRepos(repos)
}

func (g *GitlabIntegration) exportUserSourceCode(user *api.GitlabUser) (rerr error) {

	repos, err := g.exportUserRepos(user)
	if err != nil {
		rerr = err
		return
	}

	return g.exportCommonRepos(repos)
}

func (g *GitlabIntegration) exportCommonRepos(repos []*sdk.SourceCodeRepo) (rerr error) {

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
	repo.IntegrationInstanceID = g.integrationInstanceID
	if rerr = g.pipe.Write(repo); rerr != nil {
		return
	}
	g.exportRepoPullRequests(repo)
	if g.isGitlabCom {
		if rerr = g.exportRepoUsers(repo); rerr != nil {
			return
		}
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

func (g *GitlabIntegration) exportGroupWork(group *api.Group) (rerr error) {

	projects, err := g.exportGroupProjects(group)
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

func (g *GitlabIntegration) exportUserWork(user *api.GitlabUser) (rerr error) {

	projects, err := g.exportUserProjects(user)
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

func (g *GitlabIntegration) IncludeRepo(login string, name string, isArchived bool) bool {
	if g.config.Exclusions != nil && g.config.Exclusions.Matches(login, name) {
		// skip any repos that don't match our rule
		sdk.LogInfo(g.logger, "skipping repo because it matched exclusion rule", "name", name)
		return false
	}
	if g.config.Inclusions != nil && !g.config.Inclusions.Matches(login, name) {
		// skip any repos that don't match our rule
		sdk.LogInfo(g.logger, "skipping repo because it didn't match inclusion rule", "name", name)
		return false
	}
	if isArchived {
		sdk.LogInfo(g.logger, "skipping repo because it is archived", "name", name)
		return false
	}
	return true
}

func (g *GitlabIntegration) newHTTPClient(logger sdk.Logger) (url string, cl sdk.HTTPClient, err error) {

	ok, url := g.config.GetString("url")
	if !ok {
		url = "https://gitlab.com/api/v4/"
	}

	var client sdk.HTTPClient

	if g.config.APIKeyAuth != nil {
		apikey := g.config.APIKeyAuth.APIKey
		if g.config.APIKeyAuth.URL != "" {
			url = g.config.APIKeyAuth.URL
		}
		client = g.manager.HTTPManager().New(url, map[string]string{
			"Authorization": "bearer " + apikey,
		})
		sdk.LogInfo(logger, "using apikey authorization")
	} else if g.config.OAuth2Auth != nil {
		authToken := g.config.OAuth2Auth.AccessToken
		if g.config.OAuth2Auth.RefreshToken != nil {
			token, err := g.manager.AuthManager().RefreshOAuth2Token(GitlabRefType, *g.config.OAuth2Auth.RefreshToken)
			if err != nil {
				return "", nil, fmt.Errorf("error refreshing oauth2 access token: %w", err)
			}
			authToken = token
		}
		if g.config.OAuth2Auth.URL != "" {
			url = g.config.OAuth2Auth.URL
		}
		client = g.manager.HTTPManager().New(url, map[string]string{
			"Authorization": "bearer " + authToken,
		})
		sdk.LogInfo(logger, "using oauth2 authorization")
	} else if g.config.BasicAuth != nil {
		if g.config.BasicAuth.URL != "" {
			url = g.config.BasicAuth.URL
		}
		client = g.manager.HTTPManager().New(url, map[string]string{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(g.config.BasicAuth.Username+":"+g.config.BasicAuth.Password)),
		})
		sdk.LogInfo(logger, "using basic authorization", "username", g.config.BasicAuth.Username)
	} else {
		return "", nil, fmt.Errorf("supported authorization not provided. support for: apikey, oauth2, basic")
	}
	return url, client, nil
}
