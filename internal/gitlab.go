package internal

import (
	"encoding/base64"
	"fmt"
	"sync"
	"time"

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

// Enroll is called when a new integration instance is added
func (g *GitlabIntegration) Enroll(instance sdk.Instance) error {
	// attempt to add an org level web hook
	started := time.Now()
	err := g.registerWebhooks(instance)
	sdk.LogInfo(g.logger, "enroll finished", "duration", time.Since(started), "customer_id", instance.CustomerID(), "integration_instance_id", instance.IntegrationInstanceID())
	return err
}

// Dismiss is called when an existing integration instance is removed
func (g *GitlabIntegration) Dismiss(instance sdk.Instance) error {
	// TODO: Add logic for Dismiss
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

func (i *GitlabIntegration) newHTTPClient(logger sdk.Logger) (url string, cl sdk.HTTPClient, err error) {

	url = "https://gitlab.com/api/v4/"

	var client sdk.HTTPClient

	if i.config.APIKeyAuth != nil {
		apikey := i.config.APIKeyAuth.APIKey
		if i.config.APIKeyAuth.URL != "" {
			url = i.config.APIKeyAuth.URL
		}
		client = i.manager.HTTPManager().New(url, map[string]string{
			"Authorization": "bearer " + apikey,
		})
		sdk.LogInfo(logger, "using apikey authorization")
	} else if i.config.OAuth2Auth != nil {
		authToken := i.config.OAuth2Auth.AccessToken
		if i.config.OAuth2Auth.RefreshToken != nil {
			token, err := i.manager.AuthManager().RefreshOAuth2Token(gitlabRefType, *i.config.OAuth2Auth.RefreshToken)
			if err != nil {
				return "", nil, fmt.Errorf("error refreshing oauth2 access token: %w", err)
			}
			authToken = token
		}
		if i.config.OAuth2Auth.URL != "" {
			url = i.config.OAuth2Auth.URL
		}
		client = i.manager.HTTPManager().New(url, map[string]string{
			"Authorization": "bearer " + authToken,
		})
		sdk.LogInfo(logger, "using oauth2 authorization")
	} else if i.config.BasicAuth != nil {
		if i.config.BasicAuth.URL != "" {
			url = i.config.BasicAuth.URL
		}
		client = i.manager.HTTPManager().New(url, map[string]string{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(i.config.BasicAuth.Username+":"+i.config.BasicAuth.Password)),
		})
		sdk.LogInfo(logger, "using basic authorization", "username", i.config.BasicAuth.Username)
	} else {
		return "", nil, fmt.Errorf("supported authorization not provided. support for: apikey, oauth2, basic")
	}
	return url, client, nil
}
