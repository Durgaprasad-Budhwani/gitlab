package internal

import (
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

// GitlabIntegration is an integration for GitHub
type GitlabIntegration struct {
	logger  sdk.Logger
	config  sdk.Config
	manager sdk.Manager
	lock    sync.Mutex
}

var _ sdk.Integration = (*GitlabIntegration)(nil)

// Start is called when the integration is starting up
func (g *GitlabIntegration) Start(logger sdk.Logger, config sdk.Config, manager sdk.Manager) error {
	g.logger = sdk.LogWith(logger, "pkg", gitlabRefType)
	g.config = config
	g.manager = manager
	sdk.LogInfo(g.logger, "starting")
	return nil
}

const (
	// FetchAccounts will fetch accounts
	FetchAccounts = "FETCH_ACCOUNTS"
)

// Validate validate
func (g *GitlabIntegration) Validate(validate sdk.Validate) (map[string]interface{}, error) {
	config := validate.Config()

	logger := sdk.LogWith(g.logger, "customer_id", validate.CustomerID(), "integration_instance_id", validate.IntegrationInstanceID())

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
		ge.namespaceManager = NewNamespaceManager(logger, validate.State())

		accounts := []sdk.ValidatedAccount{}

		namespaces, err := api.AllNamespaces(ge.qc)
		if err != nil {
			return nil, err
		}

		sdk.LogDebug(ge.logger, "namespaces list", "namespaces", namespaces)

		for _, namespace := range namespaces {
			var repos []*sdk.SourceCodeRepo
			err = ge.fetchNamespaceProjectsRepos(namespace, func(repo *sdk.SourceCodeRepo) {
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

			// TODO: Check if Description and Visibility are strictly necessary
			err = ge.namespaceManager.SaveNamespace(*namespace)
			if err != nil {
				return nil, err
			}

			accounts = append(accounts, sdk.ValidatedAccount{
				ID:         namespace.ID,
				Name:       namespace.Name,
				AvatarURL:  namespace.AvatarURL,
				TotalCount: len(repos),
				Type:       string(accountType),
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

// Dismiss is called when an existing integration instance is removed
func (g *GitlabIntegration) Dismiss(instance sdk.Instance) error {

	// TODO: Change repos to active false
	logger := sdk.LogWith(g.logger, "customer_id", instance.CustomerID(), "integration_instance_id", instance.IntegrationInstanceID())
	started := time.Now()
	state := instance.State()
	config := instance.Config()

	sdk.LogInfo(logger, "dismiss started")

	ge, err := g.SetQueryConfig(logger, config, g.manager, instance.CustomerID())
	if err != nil {
		return fmt.Errorf("error creating http client: %w", err)
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
			sdk.LogInfo(logger, "error unregistering system webhook", "err", err)
		} else {
			sdk.LogInfo(logger, "deleted system webhook")
		}
	}

	for _, acct := range *config.Accounts {
		if acct.Type == sdk.ConfigAccountTypeOrg {
			err := wr.unregisterWebhook(sdk.WebHookScopeOrg, acct.ID, acct.ID)
			if err != nil {
				sdk.LogInfo(logger, "error unregistering namespace webhook", "err", err)
			} else {
				sdk.LogInfo(logger, "deleted namespace webhook", "id", acct.ID)
			}
		}
		customData, err := ge.namespaceManager.GetNamespace(acct.ID)
		if err != nil {
			return err
		}
		namespace := &api.Namespace{
			ID:        acct.ID,
			Name:      acct.ID,
			Path:      customData.Path,
			ValidTier: customData.ValidTier,
		}
		if acct.Type == sdk.ConfigAccountTypeOrg {
			namespace.Kind = "group"
		} else {
			namespace.Kind = "user"
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
				sdk.LogInfo(logger, "deleted repo webhook", "id", acct.ID)
			}
		}

	}

	state.Delete(ge.lastExportKey)

	sdk.LogInfo(logger, "dismiss completed", "duration", time.Since(started))

	return nil
}

// Mutation is called when a mutation is received on behalf of the integration
func (g *GitlabIntegration) Mutation(mutation sdk.Mutation) error {
	// TODO: Add mutations
	return nil
}

// Stop is called when the integration is shutting down for cleanup
func (g *GitlabIntegration) Stop() error {
	sdk.LogInfo(g.logger, "stopping")
	// TODO: Add Stop functionality
	return nil
}

func newHTTPClient(logger sdk.Logger, config sdk.Config, manager sdk.Manager) (url string, cl sdk.HTTPClient, err error) {

	url = "https://gitlab.com/api/v4/"

	if config.APIKeyAuth != nil {
		apikey := config.APIKeyAuth.APIKey
		if config.APIKeyAuth.URL != "" {
			url = sdk.JoinURL(config.APIKeyAuth.URL, "api/v4")
		}
		cl = manager.HTTPManager().New(url, map[string]string{
			"Authorization": "bearer " + apikey,
		})
		sdk.LogInfo(logger, "using apikey authorization", "apikey", apikey, "url", url)
	} else if config.OAuth2Auth != nil {
		authToken := config.OAuth2Auth.AccessToken
		if config.OAuth2Auth.URL != "" {
			url = sdk.JoinURL(config.OAuth2Auth.URL, "api/v4")
		}
		cl = manager.HTTPManager().New(url, map[string]string{
			"Authorization": "bearer " + authToken,
		})
		sdk.LogInfo(logger, "using oauth2 authorization")
	} else if config.BasicAuth != nil {
		// TODO: check if this type is supported by gitlab
		if config.BasicAuth.URL != "" {
			url = config.BasicAuth.URL
		}
		cl = manager.HTTPManager().New(url, map[string]string{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(config.BasicAuth.Username+":"+config.BasicAuth.Password)),
		})
		sdk.LogInfo(logger, "using basic authorization", "username", config.BasicAuth.Username)
	} else {
		err = fmt.Errorf("supported authorization not provided. support for: apikey, oauth2, basic")
		return
	}
	return
}
