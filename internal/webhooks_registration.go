package internal

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

func (i *GitlabIntegration) registerWebhooks(ge GitlabExport, namespaces []*api.Namespace) error {

	// TODO: Add concurrency to webhooks registration
	customerID := ge.qc.CustomerID
	integrationInstanceID := ge.integrationInstanceID
	webhookManager := i.manager.WebHookManager()

	wr := webHookRegistration{
		customerID:            customerID,
		integrationInstanceID: *integrationInstanceID,
		manager:               webhookManager,
		ge:                    &ge,
	}

	loginUser, err := api.LoginUser(ge.qc)
	if err != nil {
		return err
	}

	if !ge.isGitlabCloud && loginUser.IsAdmin {
		err = wr.registerWebhook(sdk.WebHookScopeSystem, "", "")
		if err != nil {
			sdk.LogDebug(ge.logger, "error registering sytem webhooks", "err", err)
			webhookManager.Errored(customerID, *ge.integrationInstanceID, gitlabRefType, "system", sdk.WebHookScopeSystem, err)
			return err
		}
	}

	sdk.LogDebug(ge.logger, "namespaces", "namespaces", sdk.Stringify(namespaces))
	userHasGroupWebhookAcess := make(map[string]bool)
	for _, namespace := range namespaces {
		sdk.LogDebug(ge.logger, "group webooks", "group_name", namespace.Name, "valid_tier", namespace.ValidTier)
		if namespace.Kind == "user" {
			sdk.LogDebug(ge.logger, "user namespace marked to create project webhooks", "user", namespace.Name)
			namespace.MarkedToCreateProjectWebHooks = true
			continue
		}
		if namespace.ValidTier {
			if loginUser.IsAdmin {
				err = wr.registerWebhook(sdk.WebHookScopeOrg, namespace.ID, namespace.Name)
				if err != nil {
					namespace.MarkedToCreateProjectWebHooks = true
					sdk.LogWarn(ge.logger, "there was an error trying to create namespace webhooks, will try to create project webhooks instead", "namespace", namespace.Name, "user", loginUser.Name, "user_access_level", loginUser.AccessLevel, "err", err)
				}
			} else {
				user, err := api.GroupUser(ge.qc, namespace, loginUser.StrID)
				if err != nil && strings.Contains(err.Error(), "Not found") {
					namespace.MarkedToCreateProjectWebHooks = true
					sdk.LogWarn(ge.logger, "use is not member of this namespace, will try to create project webhooks", "namespace", namespace.Name, "user_id", loginUser.RefID, "user_name", loginUser.Name, "err", err)
					continue
				}
				if err != nil {
					namespace.MarkedToCreateProjectWebHooks = true
					sdk.LogWarn(ge.logger, "there was an error trying to get namespace user access level, will try to create project webhooks instead", "namespace", namespace.Name, "user", user.Name, "user_access_level", user.AccessLevel, "err", err)
					continue
				}
				sdk.LogDebug(ge.logger, "user", "access_level", user.AccessLevel)

				if user.AccessLevel >= api.Owner {
					userHasGroupWebhookAcess[namespace.ID] = true
					err = wr.registerWebhook(sdk.WebHookScopeOrg, namespace.ID, namespace.Name)
					if err != nil {
						namespace.MarkedToCreateProjectWebHooks = true
						sdk.LogWarn(ge.logger, "there was an error trying to create namespace webhooks, will try to create project webhooks instead", "namespace", namespace.Name, "user", user.Name, "user_access_level", user.AccessLevel, "err", err)
					}
				} else {
					namespace.MarkedToCreateProjectWebHooks = true
					sdk.LogWarn(ge.logger, "at least Onwner level access is needed to create webhooks for this namespace will try to create project webhooks instead", "namespace", namespace.Name, "user", user.Name, "user_access_level", user.AccessLevel)
				}
			}

		}
	}

	sdk.LogDebug(ge.logger, "creating project webhooks")
	for _, namespace := range namespaces {
		sdk.LogDebug(ge.logger, "project webhooks", "namespace", namespace)
		if namespace.MarkedToCreateProjectWebHooks {
			ge.lastExportDate = time.Time{}
			projects, err := ge.exportNamespaceRepos(namespace)
			if err != nil {
				err = fmt.Errorf("error trying to get namespace projects err => %s", err)
				webhookManager.Errored(customerID, *integrationInstanceID, gitlabRefType, namespace.ID, sdk.WebHookScopeOrg, err)
				return err
			}
			sdk.LogDebug(ge.logger, "namespace projects", "projects", projects)
			for _, project := range projects {
				sdk.LogDebug(ge.logger, "webhook for project", "project_name", project.Name, "project_ref_id", project.RefID)
				var user *api.GitlabUser
				if userHasGroupWebhookAcess[namespace.ID] || project.OwnerRefID == loginUser.RefID {
					sdk.LogDebug(ge.logger, "registering webhook for project", "project_name", project.Name)
					err = wr.registerWebhook(sdk.WebHookScopeRepo, project.RefID, project.Name)
					if err != nil {
						err := fmt.Errorf("error trying to register project webhooks err => %s", err)
						webhookManager.Errored(customerID, *integrationInstanceID, gitlabRefType, project.ID, sdk.WebHookScopeProject, err)
						sdk.LogError(ge.logger, "error creating project webhook", "err", err)
						return err
					}
					continue
				}
				user, err := api.ProjectUser(ge.qc, project, loginUser.StrID)
				if err != nil {
					err = fmt.Errorf("error trying to get project user user => %s err => %s", loginUser.Name, err)
					webhookManager.Errored(customerID, *integrationInstanceID, gitlabRefType, project.ID, sdk.WebHookScopeProject, err)
					return err
				}
				sdk.LogDebug(ge.logger, "user project level", "user_level", user.AccessLevel)
				if user.AccessLevel >= api.Maintainer {
					sdk.LogDebug(ge.logger, "registering webhook for project", "project_name", project.Name)
					err = wr.registerWebhook(sdk.WebHookScopeRepo, project.RefID, project.Name)
					if err != nil {
						err := fmt.Errorf("error trying to register project webhooks err => %s", err)
						webhookManager.Errored(customerID, *integrationInstanceID, gitlabRefType, project.ID, sdk.WebHookScopeProject, err)
						sdk.LogError(ge.logger, "error creating project webhook", "err", err)
						return err
					}
				} else {
					err := fmt.Errorf("at least Maintainer level access is needed to create webhooks for this project project => %s user => %s user_access_level %d", project.Name, user.Name, user.AccessLevel)
					webhookManager.Errored(customerID, *integrationInstanceID, gitlabRefType, project.ID, sdk.WebHookScopeProject, err)
					sdk.LogError(ge.logger, err.Error())
				}
			}
		}
	}

	return nil
}

type webHookRegistration struct {
	manager               sdk.WebHookManager
	customerID            string
	integrationInstanceID string
	ge                    *GitlabExport
}

func (wr *webHookRegistration) registerWebhook(whType sdk.WebHookScope, entityID, entityName string) error {

	sdk.LogDebug(wr.ge.logger, "registering webhook", "type", whType, "entityID", entityID, "entityName", entityName)

	pinptWhURL, err := wr.ge.isWebHookInstalledForCurrentVersion(whType, wr.manager, wr.customerID, wr.integrationInstanceID, entityID)
	if err != nil {
		sdk.LogDebug(wr.ge.logger, "webhook already installed", "webhook_id", entityID, "type", whType)
		return nil
	}

	webHooks, err := wr.ge.getHooks(whType, entityID, entityName)
	if err != nil {
		return err
	}

	sdk.LogDebug(wr.ge.logger, "source webhooks length", "len", len(webHooks), "webhooks", webHooks)

	var found bool
	for _, wh := range webHooks {
		if strings.Contains(wh.URL, "event.api") && strings.Contains(wh.URL, "pinpoint.com") && strings.Contains(wh.URL, wr.integrationInstanceID) {
			// Check the version comming from event-api is the same
			if wh.URL == pinptWhURL {
				found = true
			} else {
				err := api.DeleteWebHook(whType, wr.ge.qc, entityID, entityName, strconv.FormatInt(wh.ID, 10))
				if err != nil {
					return err
				}
			}
		}
	}

	sdk.LogDebug(wr.ge.logger, "pinpoint webhook", "found", found)

	if !found {
		if pinptWhURL != "" {
			wr.manager.Delete(wr.customerID, wr.integrationInstanceID, gitlabRefType, entityID, whType)
		}
		params := []string{"version=" + hookVersion}
		if whType == sdk.WebHookScopeRepo {
			params = append(params, "ref_id="+entityID)
		}
		url, err := wr.manager.Create(wr.customerID, wr.integrationInstanceID, gitlabRefType, entityID, whType, params...)
		if err != nil {
			wr.manager.Delete(wr.customerID, wr.integrationInstanceID, gitlabRefType, entityID, whType)
			return err
		}
		err = api.CreateWebHook(whType, wr.ge.qc, url, entityID, entityName)
		if err != nil {
			return err
		}
		sdk.LogDebug(wr.ge.logger, "webhook created", "scope", whType, "entity_id", entityID, "entity_name", entityName)
	}

	return nil
}

func (wr *webHookRegistration) unregisterWebhook(whType sdk.WebHookScope, entityID, entityName string) error {

	webHooks, err := wr.ge.getHooks(whType, entityID, entityName)
	if err != nil {
		return err
	}

	for _, wh := range webHooks {
		if strings.Contains(wh.URL, wr.integrationInstanceID) {
			sdk.LogInfo(wr.ge.logger, "deleting webhook", "url", wh.URL)
			err = api.DeleteWebHook(whType, wr.ge.qc, entityID, entityName, strconv.FormatInt(wh.ID, 10))
			if err != nil {
				return err
			}
		}
		err := wr.manager.Delete(wr.customerID, wr.integrationInstanceID, gitlabRefType, entityID, whType)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ge *GitlabExport) isWebHookInstalledForCurrentVersion(webhookType sdk.WebHookScope, manager sdk.WebHookManager, customerID, integrationInstanceID, entityID string) (string, error) {

	pinptWebHookExist := manager.Exists(customerID, integrationInstanceID, gitlabRefType, entityID, webhookType)

	sdk.LogDebug(ge.logger, "pinpoint webhook exists", "entityID", entityID, "webhookType", webhookType, "exists", pinptWebHookExist)

	if pinptWebHookExist {
		theurl, err := manager.HookURL(customerID, integrationInstanceID, gitlabRefType, entityID, webhookType)
		if err != nil {
			return "", err
		}
		// check and see if we need to upgrade our hook
		if !strings.Contains(theurl, "version="+hookVersion) {
			sdk.LogDebug(ge.logger, "webhook version changed")
			return "", manager.Delete(customerID, integrationInstanceID, gitlabRefType, entityID, webhookType)
		}
		return theurl, nil
	}
	return "", nil
}

func (ge *GitlabExport) getHooks(webhookType sdk.WebHookScope, entityID, entityName string) (gwhs []*api.GitlabWebhook, rerr error) {
	rerr = api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (np api.NextPage, rerr error) {
		pi, whs, err := api.GetWebHooksPage(webhookType, ge.qc, entityID, entityName, params)
		if err != nil {
			return pi, err
		}
		gwhs = append(gwhs, whs...)
		return
	})
	return
}
