package api

import (
	"encoding/base64"
	"fmt"
	"github.com/pinpt/agent/v4/sdk"
)

func NewHTTPClient2(logger sdk.Logger, config sdk.Config, manager sdk.Manager) (url string, cl sdk.HTTPClient, cl2 sdk.GraphQLClient, err error) {

	url = "https://gitlab.com/api/v4/"
	graphqlURL := "https://gitlab.com/api/graphql/"

	if config.APIKeyAuth != nil {
		apikey := config.APIKeyAuth.APIKey
		if config.APIKeyAuth.URL != "" {
			url = sdk.JoinURL(config.APIKeyAuth.URL, "api/v4")
			graphqlURL = sdk.JoinURL(config.APIKeyAuth.URL, "api/graphql/")
		}
		headers := map[string]string{
			"Authorization": "bearer " + apikey,
		}
		cl = manager.HTTPManager().New(url, headers)
		cl2 = manager.GraphQLManager().New(graphqlURL, headers)
		sdk.LogInfo(logger, "using apikey authorization", "apikey", apikey, "url", url)
	} else if config.OAuth2Auth != nil {
		authToken := config.OAuth2Auth.AccessToken
		if config.OAuth2Auth.URL != "" {
			url = sdk.JoinURL(config.OAuth2Auth.URL, "api/v4")
			graphqlURL = sdk.JoinURL(config.OAuth2Auth.URL, "api/graphql/")
		}
		headers := map[string]string{
			"Authorization": "bearer " + authToken,
		}
		cl = manager.HTTPManager().New(url, headers)
		cl2 = manager.GraphQLManager().New(graphqlURL, headers)
		sdk.LogInfo(logger, "using oauth2 authorization")
	} else if config.BasicAuth != nil {
		// TODO: check if this type is supported by gitlab
		if config.BasicAuth.URL != "" {
			url = sdk.JoinURL(config.BasicAuth.URL, "api/v4")
			graphqlURL = sdk.JoinURL(config.BasicAuth.URL, "api/graphql/")
		}
		headers := map[string]string{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(config.BasicAuth.Username+":"+config.BasicAuth.Password)),
		}
		cl = manager.HTTPManager().New(url, headers)
		cl2 = manager.GraphQLManager().New(graphqlURL, headers)
		sdk.LogInfo(logger, "using basic authorization", "username", config.BasicAuth.Username)
	} else {
		err = fmt.Errorf("supported authorization not provided. support for: apikey, oauth2, basic")
		return
	}
	return
}
