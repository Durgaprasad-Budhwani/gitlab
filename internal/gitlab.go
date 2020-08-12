package internal

import (
	"encoding/base64"
	"fmt"
	"sync"

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

func (g *GitlabIntegration) Validate(config sdk.Validate) (result map[string]interface{}, err error) {
	return
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
		if config.OAuth2Auth.RefreshToken != nil {
			token, err := manager.AuthManager().RefreshOAuth2Token(gitlabRefType, *config.OAuth2Auth.RefreshToken)
			if err != nil {
				return "", nil, fmt.Errorf("error refreshing oauth2 access token: %w", err)
			}
			authToken = token
		}
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
