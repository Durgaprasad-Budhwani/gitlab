package internal

import (
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
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

// Mutation is called when a mutation is received on behalf of the integration
func (g *GitlabIntegration) Mutation(mutation sdk.Mutation) (*sdk.MutationResponse, error) {
	sdk.LogInfo(g.logger, "mutation request received", "action", mutation.Action(), "id", mutation.ID(), "customer_id", mutation.CustomerID(), "model", mutation.Model())
	// TODO: Add mutations
	return nil, nil
}

// Stop is called when the integration is shutting down for cleanup
func (g *GitlabIntegration) Stop() error {
	sdk.LogInfo(g.logger, "stopping")
	// TODO: Add Stop functionality
	return nil
}

func newHTTPClient(logger sdk.Logger, config sdk.Config, manager sdk.Manager) (url string, cl sdk.HTTPClient, cl2 sdk.GraphQLClient, err error) {

	url = "https://gitlab.com/api/v4/"
	graphqlurl := "https://gitlab.com/api/graphql/"

	if config.APIKeyAuth != nil {
		apikey := config.APIKeyAuth.APIKey
		if config.APIKeyAuth.URL != "" {
			url = sdk.JoinURL(config.APIKeyAuth.URL, "api/v4")
			graphqlurl = sdk.JoinURL(config.APIKeyAuth.URL, "api/graphql/")
		}
		headers := map[string]string{
			"Authorization": "bearer " + apikey,
		}
		cl = manager.HTTPManager().New(url, headers)
		cl2 = manager.GraphQLManager().New(graphqlurl, headers)
		sdk.LogInfo(logger, "using apikey authorization", "apikey", apikey, "url", url)
	} else if config.OAuth2Auth != nil {
		authToken := config.OAuth2Auth.AccessToken
		if config.OAuth2Auth.URL != "" {
			url = sdk.JoinURL(config.OAuth2Auth.URL, "api/v4")
			graphqlurl = sdk.JoinURL(config.OAuth2Auth.URL, "api/graphql/")
		}
		headers := map[string]string{
			"Authorization": "bearer " + authToken,
		}
		cl = manager.HTTPManager().New(url, headers)
		cl2 = manager.GraphQLManager().New(graphqlurl, headers)
		sdk.LogInfo(logger, "using oauth2 authorization")
	} else if config.BasicAuth != nil {
		// TODO: check if this type is supported by gitlab
		if config.BasicAuth.URL != "" {
			url = sdk.JoinURL(config.BasicAuth.URL, "api/v4")
			graphqlurl = sdk.JoinURL(config.BasicAuth.URL, "api/graphql/")
		}
		headers := map[string]string{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(config.BasicAuth.Username+":"+config.BasicAuth.Password)),
		}
		cl = manager.HTTPManager().New(url, headers)
		cl2 = manager.GraphQLManager().New(graphqlurl, headers)
		sdk.LogInfo(logger, "using basic authorization", "username", config.BasicAuth.Username)
	} else {
		err = fmt.Errorf("supported authorization not provided. support for: apikey, oauth2, basic")
		return
	}
	return
}
