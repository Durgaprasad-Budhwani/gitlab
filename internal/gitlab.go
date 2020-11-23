package internal

import (
	"fmt"
	"sync"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

// GitlabIntegration is an integration for GitHub
type GitlabIntegration struct {
	config  sdk.Config
	manager sdk.Manager
	lock    sync.Mutex
}

var _ sdk.Integration = (*GitlabIntegration)(nil)

// Start is called when the integration is starting up
func (g *GitlabIntegration) Start(logger sdk.Logger, config sdk.Config, manager sdk.Manager) error {
	g.config = config
	g.manager = manager
	sdk.LogInfo(logger, "starting")
	return nil
}

const (
	// FetchAccounts will fetch accounts
	FetchAccounts = "FETCH_ACCOUNTS"
)

// Validate validate
func (g *GitlabIntegration) Validate(validate sdk.Validate) (map[string]interface{}, error) {
	config := validate.Config()

	logger := validate.Logger()

	found, action := config.GetString("action")
	if !found {
		return nil, fmt.Errorf("validation had no action")
	}
	switch action {
	case FetchAccounts:

		ge, err := g.SetQueryConfig(logger, config, g.manager, validate.CustomerID())
		if err != nil {
			return nil, err
		}
		ge.qc.WorkManager = NewWorkManager(logger, validate.State())

		accounts := []sdk.ValidatedAccount{}

		namespaces, err := api.AllNamespaces(ge.qc)
		if err != nil {
			return nil, err
		}

		sdk.LogDebug(ge.logger, "namespaces list", "namespaces", namespaces)

		for _, namespace := range namespaces {
			var repos []*api.GitlabProjectInternal
			err = ge.fetchNamespaceProjectsRepos(namespace, func(repo *api.GitlabProjectInternal) {
				repos = append(repos, repo)
			})
			if err != nil {
				return nil, err
			}

			var accountType sdk.ConfigAccountType
			if namespace.Kind == "group" {
				accountType = sdk.ConfigAccountTypeOrg
			} else {
				accountType = sdk.ConfigAccountTypeUser
			}

			accounts = append(accounts, sdk.ValidatedAccount{
				ID:         namespace.ID,
				Name:       namespace.Name,
				AvatarURL:  namespace.AvatarURL,
				TotalCount: len(repos),
				Type:       string(accountType),
				Selected:   true,
			})
		}

		return map[string]interface{}{
			"accounts": accounts,
		}, nil
	default:
		return nil, fmt.Errorf("unknown action %s", action)
	}
}

// Enroll is called when a new integration instance is added
func (g *GitlabIntegration) Enroll(instance sdk.Instance) error {
	// attempt to add an org level web hook
	// started := time.Now()
	// sdk.LogInfo(g.logger, "enroll finished", "duration", time.Since(started), "customer_id", instance.CustomerID(), "integration_instance_id", instance.IntegrationInstanceID())
	return nil
}

// Stop is called when the integration is shutting down for cleanup
func (g *GitlabIntegration) Stop(logger sdk.Logger) error {
	sdk.LogInfo(logger, "stopping")
	// TODO: Add Stop functionality
	return nil
}
