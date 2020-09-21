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
	historical                 bool
	state                      sdk.State
	config                     sdk.Config
	lastExportDate             time.Time
	lastExportDateGitlabFormat string
	isGitlabCloud              bool
	integrationInstanceID      *string
	lastExportKey              string
	systemWebHooksEnabled      bool
}

const concurrentAPICalls = 10

func (i *GitlabExport) workConfig() error {

	wc := &sdk.WorkConfig{}
	wc.ID = sdk.NewWorkConfigID(i.qc.CustomerID, "gitlab", *i.integrationInstanceID)
	wc.CreatedAt = sdk.EpochNow()
	wc.UpdatedAt = sdk.EpochNow()
	wc.CustomerID = i.qc.CustomerID
	wc.IntegrationInstanceID = *i.integrationInstanceID
	wc.RefType = "gitlab"
	wc.Statuses = sdk.WorkConfigStatuses{
		OpenStatus:       []string{"open", "Open"},
		InProgressStatus: []string{"in progress", "In progress", "In Progress"},
		ClosedStatus:     []string{"closed", "Closed"},
	}

	return i.pipe.Write(wc)
}

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
	ge.qc.BaseURL = u.Scheme + "://" + u.Hostname()
	sdk.LogDebug(logger, "base url", "base-url", ge.qc.BaseURL)
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
	ge.qc.WorkManager = NewWorkManager(logger)
	ge.qc.SprintManager = NewSprintManager(logger)
	ge.qc.IntegrationInstanceID = *ge.integrationInstanceID
	ge.qc.UserManager = NewUserManager(ge.qc.CustomerID, export, ge.state, ge.pipe, ge.qc.IntegrationInstanceID)
	ge.qc.Pipe = ge.pipe

	ge.lastExportKey = "last_export_date"

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

	exportStartDate := time.Now()

	err = gexport.workConfig()
	if err != nil {
		return err
	}

	sdk.LogDebug(logger, "accounts", "accounts", config.Accounts)

	allnamespaces := make([]*api.Namespace, 0)
	if config.Accounts == nil {
		namespaces, err := api.AllNamespaces(gexport.qc)
		if err != nil {
			return err
		}
		allnamespaces = append(allnamespaces, namespaces...)
	} else {

		namespaces, err := getNamespacesSelectedAccounts(gexport.qc, config.Accounts)
		if err != nil {
			sdk.LogError(logger, "error getting data accounts", "err", err)
		}

		allnamespaces = append(allnamespaces, namespaces...)
	}

	sdk.LogInfo(logger, "registering webhooks")

	// err = i.registerWebhooks(gexport, allnamespaces)
	// if err != nil {
	// 	return err
	// }

	sdk.LogInfo(logger, "registering webhooks done")

	for _, namespace := range allnamespaces {
		sdk.LogDebug(logger, "namespace", "name", namespace.Name)
		projectUsersMap := make(map[string]api.UsernameMap)

		repos, err := gexport.exportNamespaceSourceCode(namespace, projectUsersMap)
		if err != nil {
			sdk.LogWarn(logger, "error exporting sourcecode namespace", "namespace_id", namespace.ID, "namespace_name", namespace.Name, "err", err)
		}

		err = gexport.exportReposWork(repos, projectUsersMap)
		if err != nil {
			sdk.LogWarn(logger, "error exporting work repos", "namespace_id", namespace.ID, "namespace_name", namespace.Name, "err", err)
		}

		reposSprints, err := gexport.fetchProjectsSprints(repos)
		if err != nil {
			sdk.LogWarn(logger, "error fetching repo sprints ", "namespace_id", namespace.ID, "namespace_name", namespace.Name, "err", err)
		}

		groupSprints, err := gexport.fetchGroupSprints(namespace)
		if err != nil {
			sdk.LogWarn(logger, "error fetching group sprints ", "namespace_id", namespace.ID, "namespace_name", namespace.Name, "err", err)
		}

		err = gexport.exportGroupBoards(namespace, repos)
		if err != nil {
			return err
		}
		err = gexport.exportReposBoards(repos)
		if err != nil {
			return err
		}

		if err := gexport.exportSprints(append(reposSprints, groupSprints...)); err != nil {
			return err
		}
		if err := gexport.exportSprints(reposSprints); err != nil {
			return err
		}
	}

	return gexport.state.Set(gexport.lastExportKey, exportStartDate.Format(time.RFC3339))
}

func (ge *GitlabExport) exportNamespaceSourceCode(namespace *api.Namespace, projectUsersMap map[string]api.UsernameMap) ([]*sdk.SourceCodeRepo, error) {

	if !ge.isGitlabCloud {
		if err := ge.exportEnterpriseUsers(); err != nil {
			return nil, fmt.Errorf("error on export enterprise users %s", err)
		}
	}

	repos, err := ge.exportNamespaceRepos(namespace)
	if err != nil {
		return nil, fmt.Errorf("error on export namespace repos %s", err)
	}

	return repos, ge.exportCommonRepos(repos, projectUsersMap)
}

func (ge *GitlabExport) exportCommonRepos(repos []*sdk.SourceCodeRepo, projectUsersMap map[string]api.UsernameMap) error {

	for _, repo := range repos {
		err := ge.exportRepoAndWrite(repo, projectUsersMap)
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

func (ge *GitlabExport) exportRepoAndWrite(repo *sdk.SourceCodeRepo, projectUsersMap map[string]api.UsernameMap) error {
	repo.IntegrationInstanceID = ge.integrationInstanceID
	if err := ge.pipe.Write(repo); err != nil {
		return err
	}
	if err := ge.pipe.Write(ToProject(repo)); err != nil {
		return err
	}
	ge.exportRepoPullRequests(repo)
	if ge.isGitlabCloud {
		users, err := ge.exportRepoUsers(repo)
		if err != nil {
			return err
		}
		projectUsersMap[repo.RefID] = users
	}
	return nil
}

func (ge *GitlabExport) exportProjectAndWrite(project *sdk.SourceCodeRepo, users api.UsernameMap) error {
	ge.exportProjectIssues(project, users)
	return nil
}

func (ge *GitlabExport) exportReposWork(projects []*sdk.SourceCodeRepo, projectUsersMap map[string]api.UsernameMap) (rerr error) {

	for _, project := range projects {
		err := ge.exportProjectAndWrite(project, projectUsersMap[project.RefID])
		if err != nil {
			sdk.LogError(ge.logger, "error exporting project", "project", project.Name, "project_refid", project.RefID, "err", err)
		}
	}
	rerr = ge.pipe.Flush()
	if rerr != nil {
		return
	}

	sdk.LogDebug(ge.logger, "remaining project issues")

	for _, project := range projects {
		ge.exportRemainingProjectIssues(project, projectUsersMap[project.RefID])
	}

	return
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
