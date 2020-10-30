package internal

import (
	"fmt"
	"time"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

// Dismiss is called when an existing integration instance is removed
func (g *GitlabIntegration) Dismiss(instance sdk.Instance) error {

	logger := sdk.LogWith(instance.Logger(), "event", "dismiss")
	started := time.Now()
	state := instance.State()
	pipe := instance.Pipe()
	config := instance.Config()

	sdk.LogInfo(logger, "dismiss started")

	ge, err := g.SetQueryConfig(logger, config, g.manager, instance.CustomerID())
	if err != nil {
		return fmt.Errorf("error creating http client: %w", err)
	}
	ge.repoProjectManager = NewRepoProjectManager(logger, state, pipe)

	err = ge.repoProjectManager.DeactivateReposAndProjects()
	if err != nil {
		return err
	}

	wr := webHookRegistration{
		customerID:            instance.CustomerID(),
		integrationInstanceID: instance.IntegrationInstanceID(),
		manager:               g.manager.WebHookManager(),
		ge:                    &ge,
	}

	loginUser, err := api.LoginUser(ge.qc)
	if err != nil {
		return fmt.Errorf("error getting user info %w", err)
	}

	if !ge.isGitlabCloud && loginUser.IsAdmin {
		err := wr.unregisterWebhook(sdk.WebHookScopeSystem, "", "")
		if err != nil {
			sdk.LogError(logger, "error unregistering system webhook", "err", err)
		} else {
			sdk.LogInfo(logger, "deleted system webhook")
		}
	}

	namespaces, err := getNamespacesSelectedAccounts(ge.qc, config.Accounts)
	if err != nil {
		sdk.LogError(logger, "error getting data accounts", "err", err)
	}

	for _, namespace := range namespaces {
		if namespace.Kind == "group" {
			err := wr.unregisterWebhook(sdk.WebHookScopeOrg, namespace.ID, namespace.ID)
			if err != nil {
				sdk.LogInfo(logger, "error unregistering namespace webhook", "err", err)
			} else {
				sdk.LogInfo(logger, "deleted namespace webhook", "id", namespace.ID)
			}
		}
		var repos []*sdk.SourceCodeRepo
		err = ge.fetchNamespaceProjectsRepos(namespace, func(repo *sdk.SourceCodeRepo) {
			repos = append(repos, repo)
		})
		if err != nil {
			return err
		}
		for _, repo := range repos {
			err := wr.unregisterWebhook(sdk.WebHookScopeRepo, repo.RefID, repo.Name)
			if err != nil {
				sdk.LogInfo(logger, "error unregistering repo webhook", "err", err)
			} else {
				sdk.LogInfo(logger, "deleted repo webhook", "id", namespace.ID)
			}
		}
		sdk.LogInfo(logger, "deleted namespace state", "id", namespace.ID)
	}

	err = state.Delete(ge.lastExportKey)
	if err != nil {
		return err
	}

	sdk.LogInfo(logger, "dismiss completed", "duration", time.Since(started))

	return nil
}
