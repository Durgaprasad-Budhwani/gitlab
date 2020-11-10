package internal

import (
	"fmt"
	"net/url"
	"time"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
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
	repoProjectManager         *RepoProjectManager
}

const concurrentAPICalls = 10

func (i *GitlabIntegration) SetQueryConfig(logger sdk.Logger, config sdk.Config, manager sdk.Manager, customerID string) (ge GitlabExport, rerr error) {

	apiURL, client, graphql, err := newHTTPClient(logger, config, manager)
	if err != nil {
		rerr = err
		return
	}

	r := api.NewRequester(logger, client, concurrentAPICalls)
	ge.qc.Get = r.Get
	ge.qc.Post = r.Post
	ge.qc.Delete = r.Delete
	ge.qc.Put = r.Put
	ge.qc.Logger = logger
	ge.qc.RefType = gitlabRefType
	ge.qc.CustomerID = customerID
	ge.qc.RefType = gitlabRefType
	ge.qc.GraphRequester = api.NewGraphqlRequester(graphql, concurrentAPICalls, logger)
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
	ge.qc.WorkManager = NewWorkManager(logger, ge.state)
	ge.qc.IntegrationInstanceID = *ge.integrationInstanceID
	ge.qc.UserManager = NewUserManager(ge.qc.CustomerID, export, ge.state, ge.pipe, ge.qc.IntegrationInstanceID)
	ge.qc.Pipe = ge.pipe
	ge.qc.State = ge.state
	ge.repoProjectManager = NewRepoProjectManager(logger, ge.state, ge.pipe)

	ge.lastExportKey = "last_export_date"

	if rerr = ge.exportDate(); rerr != nil {
		return
	}

	ge.qc.Historical = ge.historical

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
			ge.historical = true
			return
		}
		lastExportDate, err := time.Parse(time.RFC3339, exportDate)
		if err != nil {
			rerr = fmt.Errorf("error formating last export date. date %s err %s", exportDate, err)
			return
		}

		ge.lastExportDate = lastExportDate.UTC()
		ge.lastExportDateGitlabFormat = lastExportDate.UTC().Format(api.GitLabDateTimeFormat)
	}

	sdk.LogDebug(ge.logger, "last export date", "date", ge.lastExportDate)

	return
}

// GitlabRefType Integraion constant type
const gitlabRefType = "gitlab"

// Export is called to tell the integration to run an export
func (i *GitlabIntegration) Export(export sdk.Export) error {
	logger := sdk.LogWith(export.Logger(), "job_id", export.JobID())

	sdk.LogInfo(logger, "export started", "historical", export.Historical())

	config := export.Config()

	gexport, err := gitlabExport(i, logger, export)
	if err != nil {
		return err
	}

	exportStartDate := time.Now()

	err = gexport.workConfig()
	if err != nil {
		return err
	}

	allnamespaces := make([]*api.Namespace, 0)
	if config.Accounts == nil {
		namespaces, err := api.AllNamespaces(gexport.qc)
		if err != nil {
			sdk.LogError(logger, "error getting data accounts", "err", err)
			return err
		}
		allnamespaces = append(allnamespaces, namespaces...)
	} else {
		namespaces, err := getNamespacesSelectedAccounts(gexport.qc, config.Accounts)
		if err != nil {
			sdk.LogError(logger, "error getting data accounts", "err", err)
			return err
		}
		allnamespaces = append(allnamespaces, namespaces...)
	}

	sdk.LogInfo(logger, "registering webhooks", "config", sdk.Stringify(config))

	err = i.registerWebhooks(gexport, allnamespaces)
	if err != nil {
		sdk.LogError(logger, "error registering webhooks", "err", err)
	}

	sdk.LogInfo(logger, "registering webhooks done")

	if gexport.historical {
		sdk.LogInfo(logger, "deleting work manager state")
		if err := gexport.qc.WorkManager.Delete(); err != nil {
			sdk.LogError(logger, "error deleting work manager state", "err", err)
			return err
		}
	} else {
		sdk.LogInfo(logger, "recovering work manager state")
		if err := gexport.qc.WorkManager.Restore(); err != nil {
			sdk.LogError(logger, "error recovering work manager state", "err", err)
			return err
		}
	}

	validServerVersion := true

	if !gexport.isGitlabCloud {
		validServerVersion, err = gexport.ValidServerVersion()
		if err != nil {
			sdk.LogError(logger, "error recovering work manager state", "err", err)
			return err
		}
		if !validServerVersion {
			sdk.LogWarn(logger, "invalid gitlab version, skipping work processing")
		}
	}

	for _, namespace := range allnamespaces {
		l := sdk.LogWith(logger, "namespace_id", namespace.ID, "namespace_name", namespace.Name)
		projectUsersMap := make(map[string]api.UsernameMap)
		repos, err := gexport.exportNamespaceSourceCode(namespace, projectUsersMap)
		if err != nil {
			sdk.LogError(logger, "error exporting sourcecode namespace", "err", err)
			return err
		}

		if validServerVersion {

			if err := api.CreateHelperSprintToUnsetIssues(gexport.qc, namespace); err != nil {
				return err
			}

			err = gexport.exportProjectsWork(repos, projectUsersMap)
			if err != nil {
				sdk.LogWarn(l, "error exporting work repos", "err", err)
				return err
			}

			if len(repos) > 0 {
				if err := gexport.exportEpics(namespace, repos, projectUsersMap); err != nil {
					sdk.LogWarn(l, "error exporting repos epics", "err", err)
					return err
				}
			}

			err = gexport.exportProjectsMilestones(repos)
			if err != nil {
				sdk.LogWarn(l, "error exporting repos milestones", "err", err)
				return err
			}

			err = gexport.exportGroupMilestones(namespace, repos)
			if err != nil {
				sdk.LogWarn(l, "error exporting group milestones", "err", err)
				return err
			}

			sprints, err := gexport.fetchGroupSprints(namespace)
			if err != nil {
				sdk.LogWarn(l, "error fetching group sprints", "err", err)
				return err
			}

			err = gexport.exportGroupBoards(namespace, repos)
			if err != nil {
				sdk.LogWarn(l, "error exporting group boards", "err", err)
				return err
			}
			err = gexport.exportReposBoards(repos)
			if err != nil {
				sdk.LogWarn(l, "error exporting repos boards", "err", err)
				return err
			}

			if err := gexport.exportSprints(sprints); err != nil {
				sdk.LogWarn(l, "error exporting group sprints", "err", err)
				return err
			}
		}
	}
	err = gexport.repoProjectManager.PersistRepos()
	if err != nil {
		return err
	}

	sdk.LogInfo(logger, "persisting work manager into state")
	if err := gexport.qc.WorkManager.Persist(); err != nil {
		sdk.LogError(logger, "error persisting work manager state", "err", err)
		return err
	}

	return gexport.state.Set(gexport.lastExportKey, exportStartDate.Format(time.RFC3339))
}

func (ge *GitlabExport) exportNamespaceSourceCode(namespace *api.Namespace, projectUsersMap map[string]api.UsernameMap) ([]*api.GitlabProjectInternal, error) {

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

func (ge *GitlabExport) exportCommonRepos(repos []*api.GitlabProjectInternal, projectUsersMap map[string]api.UsernameMap) error {

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

func (ge *GitlabExport) exportRepoAndWrite(repo *api.GitlabProjectInternal, projectUsersMap map[string]api.UsernameMap) error {
	repo.IntegrationInstanceID = ge.integrationInstanceID
	if err := ge.pipe.Write(repo); err != nil {
		return err
	}
	p := ToProject(repo)
	if err := ge.pipe.Write(p); err != nil {
		return err
	}
	if err := ge.writeProjectCapacity(p); err != nil {
		return err
	}
	ge.repoProjectManager.AddRepo(repo)
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

func (ge *GitlabExport) exportProjectAndWrite(project *api.GitlabProjectInternal, users api.UsernameMap) error {
	ge.exportProjectIssues(project, users)
	return nil
}

func (ge *GitlabExport) exportProjectsWork(projects []*api.GitlabProjectInternal, projectUsersMap map[string]api.UsernameMap) (rerr error) {

	sdk.LogDebug(ge.logger, "exporting projects issues")

	for _, project := range projects {
		ge.exportProjectIssues(project, projectUsersMap[project.RefID])
	}

	return
}

func (ge *GitlabExport) IncludeRepo(namespaceID string, name string, isArchived bool) bool {
	if ge.config.Exclusions != nil && ge.config.Exclusions.Matches(namespaceID, name) {
		// skip any repos that don't match our rule
		sdk.LogInfo(ge.logger, "skipping repo because it matched exclusion rule", "name", name)
		return false
	}
	if ge.config.Inclusions != nil && !ge.config.Inclusions.Matches(namespaceID, name) {
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
