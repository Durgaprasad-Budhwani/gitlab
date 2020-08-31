package internal

import (
	"fmt"
	"net/url"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

type GitlabExport struct {
	logger                     sdk.Logger
	qc                         api.QueryContext
	pipe                       sdk.Pipe
	isssueFutures              []IssueFuture
	historical                 bool
	state                      sdk.State
	config                     sdk.Config
	lastExportDate             time.Time
	lastExportDateGitlabFormat string
	isGitlabCloud              bool
	integrationInstanceID      *string
	integrationType            string
	lastExportKey              string
	systemWebHooksEnabled      bool
}

const concurrentAPICalls = 10

func (i *GitlabIntegration) SetQueryConfig(logger sdk.Logger, config sdk.Config, manager sdk.Manager, customerID string) (ge GitlabExport, rerr error) {

	apiURL, client, err := newHTTPClient(logger, config, manager)
	if err != nil {
		rerr = err
		return
	}

	r := api.NewRequester(logger, client, concurrentAPICalls)
	ge.qc.Get = r.Get
	ge.qc.Post = r.Post
	ge.qc.Delete = r.Delete
	ge.qc.Logger = logger
	ge.qc.RefType = gitlabRefType
	ge.qc.CustomerID = customerID
	ge.logger = logger

	u, err := url.Parse(apiURL)
	if err != nil {
		rerr = fmt.Errorf("url is not valid: %v", err)
		return
	}
	ge.isGitlabCloud = u.Hostname() == "gitlab.com"

	return ge, nil
}

func gitlabExport(i *GitlabIntegration, logger sdk.Logger, export sdk.Export) (ge GitlabExport, rerr error) {

	// TODO: Add logic for incrementals
	// to get users and repos details
	// if there is not system hook available
	ge, rerr = i.SetQueryConfig(logger, export.Config(), i.manager, export.CustomerID())
	if rerr != nil {
		return
	}

	ge.pipe = export.Pipe()
	ge.historical = export.Historical()
	ge.state = export.State()
	ge.config = export.Config()
	ge.integrationInstanceID = sdk.StringPointer(export.IntegrationInstanceID())

	// TODO: call WORK export
	ge.integrationType = IntegrationSourceCodeType

	ge.lastExportKey = string(ge.integrationType) + "@last_export_date"

	if rerr = ge.exportDate(); rerr != nil {
		return
	}

	return
}

func (ge *GitlabExport) exportDate() (rerr error) {

	if !ge.historical {
		var exportDate string
		ok, err := ge.state.Get(ge.lastExportKey, &exportDate)
		if err != nil {
			rerr = err
			return
		}
		if !ok {
			return
		}
		lastExportDate, err := time.Parse(time.RFC3339, exportDate)
		if err != nil {
			rerr = fmt.Errorf("error formating last export date. date %s err %s", exportDate, err)
			return
		}

		ge.lastExportDate = lastExportDate.UTC()
		ge.lastExportDateGitlabFormat = lastExportDate.UTC().Format(GitLabDateFormat)
	}

	sdk.LogDebug(ge.logger, "last export date", "date", ge.lastExportDate)

	return
}

// Integration constants types
const (
	IntegrationSourceCodeType = "SOURCECODE"
	IntegrationWorkType       = "WORK"
)

// GitlabRefType Integraion constant type
const gitlabRefType = "gitlab"

// GitLabDateFormat gitlab layout to format dates
const GitLabDateFormat = "2006-01-02T15:04:05.000Z"

// Export is called to tell the integration to run an export
func (i *GitlabIntegration) Export(export sdk.Export) error {

	logger := sdk.LogWith(i.logger, "customer_id", export.CustomerID(), "job_id", export.JobID())

	sdk.LogInfo(logger, "export started", "historical", export.Historical())

	config := export.Config()

	// TODO: Create a list with the most common use cases, prioritize them and work on them
	// For instance: It is higher priority to have SOURCECODE ready first than WORK

	// TODO: remove webhooks in case inclusions/exclusions change

	gexport, err := gitlabExport(i, logger, export)
	if err != nil {
		return err
	}

	sdk.LogInfo(logger, "integraion type", "type", gexport.integrationType)

	sdk.LogInfo(logger, "registering webhooks")

	err = i.registerWebhooks(gexport)
	if err != nil {
		return err
	}

	sdk.LogInfo(logger, "registering webhooks done")

	exportStartDate := time.Now()

	orgs := make([]*api.Group, 0)
	users := make([]*api.GitlabUser, 0)
	if config.Accounts == nil {
		groups, err := api.GroupsAll(gexport.qc)
		if err != nil {
			return err
		}
		orgs = append(orgs, groups...)

		user, err := api.LoginUser(gexport.qc)
		if err != nil {
			return err
		}
		users = append(users, user)
	} else {
		for _, acct := range *config.Accounts {
			if acct.Type == sdk.ConfigAccountTypeOrg {
				orgs = append(orgs, &api.Group{ID: acct.ID})
			} else {
				users = append(users, &api.GitlabUser{StrID: acct.ID})
			}
		}
	}

	for _, group := range orgs {
		sdk.LogDebug(logger, "group", "name", group.Name)
		switch gexport.integrationType {
		case IntegrationSourceCodeType:
			err := gexport.exportGroupSourceCode(group)
			if err != nil {
				sdk.LogWarn(logger, "error exporting sourcecode group", "group_id", group.ID, "group_name", group.Name, "err", err)
			}
		case IntegrationWorkType:
			err := gexport.exportGroupWork(group)
			if err != nil {
				sdk.LogWarn(logger, "error exporting sourcecode group", "group_id", group.ID, "group_name", group.Name, "err", err)
			}
		default:
			return fmt.Errorf("integration type not defined %s", gexport.integrationType)
		}
	}

	for _, user := range users {
		sdk.LogDebug(logger, "user", "name", user.Name)
		switch gexport.integrationType {
		case IntegrationSourceCodeType:
			if err := gexport.exportUserSourceCode(user); err != nil {
				sdk.LogWarn(logger, "error exporting work user", "user_id", user.ID, "user_name", user.Name, "err", err)
			}
		case IntegrationWorkType:
			if err := gexport.exportUserWork(user); err != nil {
				sdk.LogWarn(logger, "error exporting work user", "user_id", user.ID, "user_name", user.Name, "err", err)
			}
		default:
			return fmt.Errorf("integration type not defined %s", gexport.integrationType)
		}
	}

	return gexport.state.Set(gexport.lastExportKey, exportStartDate.Format(time.RFC3339))
}

func (ge *GitlabExport) exportGroupSourceCode(group *api.Group) error {

	if !ge.isGitlabCloud {
		if err := ge.exportEnterpriseUsers(); err != nil {
			return err
		}
	}

	repos, err := ge.exportGroupRepos(group)
	if err != nil {
		return err
	}

	return ge.exportCommonRepos(repos)
}

func (ge *GitlabExport) exportUserSourceCode(user *api.GitlabUser) error {

	repos, err := ge.exportUserRepos(user)
	if err != nil {
		return err
	}

	return ge.exportCommonRepos(repos)
}

func (ge *GitlabExport) exportCommonRepos(repos []*sdk.SourceCodeRepo) error {

	for _, repo := range repos {
		err := ge.exportRepoAndWrite(repo)
		if err != nil {
			sdk.LogError(ge.logger, "error exporting repo", "repo", repo.Name, "repo_refid", repo.RefID, "err", err)
		}
	}

	if err := ge.pipe.Flush(); err != nil {
		return err
	}

	for _, repo := range repos {
		ge.exportRemainingRepoPullRequests(repo)
	}

	return nil
}

func (ge *GitlabExport) exportRepoAndWrite(repo *sdk.SourceCodeRepo) error {
	repo.IntegrationInstanceID = ge.integrationInstanceID
	if err := ge.pipe.Write(repo); err != nil {
		return err
	}
	ge.exportRepoPullRequests(repo)
	if ge.isGitlabCloud {
		if err := ge.exportRepoUsers(repo); err != nil {
			return err
		}
	}
	return nil
}

func (ge *GitlabExport) exportProjectAndWrite(project *sdk.WorkProject, projectUsersMap map[string]api.UsernameMap) error {
	project.IntegrationInstanceID = ge.integrationInstanceID
	if err := ge.pipe.Write(project); err != nil {
		return err
	}
	users, err := ge.exportProjectUsers(project)
	if err != nil {
		return err
	}
	projectUsersMap[project.RefID] = users
	ge.exportProjectIssues(project, users)
	if err := ge.exportProjectSprints(project); err != nil {
		return err
	}
	return nil
}

func (ge *GitlabExport) exportIssuesFutures(projectUsersMap map[string]api.UsernameMap) {

	sdk.LogDebug(ge.logger, "remaining issues", "futures count", len(ge.isssueFutures))

	for _, f := range ge.isssueFutures {
		ge.exportRemainingProjectIssues(f.Project, projectUsersMap[f.Project.RefID])
	}

}

func (ge *GitlabExport) exportGroupWork(group *api.Group) (rerr error) {

	projects, err := ge.exportGroupProjects(group)
	if err != nil {
		rerr = err
		return
	}
	projectUsersMap := make(map[string]api.UsernameMap)
	for _, project := range projects {
		err := ge.exportProjectAndWrite(project, projectUsersMap)
		if err != nil {
			sdk.LogError(ge.logger, "error exporting project", "project", project.Name, "project_refid", project.RefID, "err", err)
		}
	}
	rerr = ge.pipe.Flush()
	if rerr != nil {
		return
	}
	sdk.LogDebug(ge.logger, "remaining project issues", "futures count", len(ge.isssueFutures))
	ge.exportIssuesFutures(projectUsersMap)

	return
}

func (ge *GitlabExport) exportUserWork(user *api.GitlabUser) error {

	projects, err := ge.exportUserProjects(user)
	if err != nil {
		return err
	}
	projectUsersMap := make(map[string]api.UsernameMap)
	for _, project := range projects {
		err := ge.exportProjectAndWrite(project, projectUsersMap)
		if err != nil {
			sdk.LogError(ge.logger, "error exporting project", "project", project.Name, "project_refid", project.RefID, "err", err)
		}
	}
	err = ge.pipe.Flush()
	if err != nil {
		return err
	}
	sdk.LogDebug(ge.logger, "remaining project issues", "futures count", len(ge.isssueFutures))
	ge.exportIssuesFutures(projectUsersMap)

	return nil
}

func (ge *GitlabExport) IncludeRepo(login string, name string, isArchived bool) bool {
	if ge.config.Exclusions != nil && ge.config.Exclusions.Matches(login, name) {
		// skip any repos that don't match our rule
		sdk.LogInfo(ge.logger, "skipping repo because it matched exclusion rule", "name", name)
		return false
	}
	if ge.config.Inclusions != nil && !ge.config.Inclusions.Matches(login, name) {
		// skip any repos that don't match our rule
		sdk.LogInfo(ge.logger, "skipping repo because it didn't match inclusion rule", "name", name)
		return false
	}
	if isArchived {
		sdk.LogInfo(ge.logger, "skipping repo because it is archived", "name", name)
		return false
	}
	return true
}
