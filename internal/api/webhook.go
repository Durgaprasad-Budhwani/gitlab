package api

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pinpt/agent.next/sdk"
)

var systemEventNames = []string{
	"project_destroy",
	"project_rename",
	"project_update",
	"user_add_to_team",
	"user_remove_from_team",
	"user_update_for_team",
	"user_create",
	"user_destroy",
	"user_rename",
	"group_destroy",
	"group_rename",
	"user_add_to_group",
	"user_remove_from_group",
	"user_update_for_group",
}

var webHookParams map[sdk.WebHookScope]url.Values = map[sdk.WebHookScope]url.Values{
	sdk.WebHookScopeSystem: {
		"repository_update_events": []string{"true"},
	},
	sdk.WebHookScopeOrg: {
		"merge_requests_events": []string{"true"},
		"note_events":           []string{"true"},
	},
	sdk.WebHookScopeRepo: {
		"merge_requests_events": []string{"true"},
		"note_events":           []string{"true"},
	},
}

// CreateWebHook create
func CreateWebHook(whType sdk.WebHookScope, qc QueryContext, eventAPIWebhookURL, entityID, entityName string) error {

	sdk.LogInfo(qc.Logger, fmt.Sprintf("create %s webhooks", whType), "entityID", entityID, "entityName", entityName)

	objectPath := buildPath(whType, entityID)

	params := webHookParams[whType]
	params.Set("url", eventAPIWebhookURL)
	params.Set("enable_ssl_verification", "true")

	var resp interface{}

	_, err := qc.Post(objectPath, params, strings.NewReader(""), &resp)
	if err != nil {
		return err
	}

	return nil
}

// GitlabWebhook webhook object
type GitlabWebhook struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

// GetWebHooksPage get web-hooks page
func GetWebHooksPage(webhookType sdk.WebHookScope, qc QueryContext, entityID string, entityName string, params url.Values) (page NextPage, gwhs []*GitlabWebhook, err error) {

	sdk.LogDebug(qc.Logger, fmt.Sprintf("%s webhooks page", webhookType), "entityID", entityID, "entityName", entityName)

	objectPath := buildPath(webhookType, entityID)

	page, err = qc.Get(objectPath, params, &gwhs)

	return
}

func buildPath(whType sdk.WebHookScope, entityID string) string {

	var path string

	switch whType {
	case sdk.WebHookScopeSystem:
		path = "hooks"
	case sdk.WebHookScopeOrg:
		path = sdk.JoinURL("groups", entityID, "hooks")
	case sdk.WebHookScopeRepo:
		path = sdk.JoinURL("projects", entityID, "hooks")
	}

	return path
}

// DeleteWebHook delete webhook
func DeleteWebHook(whType sdk.WebHookScope, qc QueryContext, entityID, entityName, whID string) error {

	sdk.LogInfo(qc.Logger, fmt.Sprintf("delete %s webhook", whType), "entityID", entityID, "entityName", entityName, "webhookID", whID)

	objectPath := buildDeletePath(whType, entityID, whID)

	var resp interface{}

	_, err := qc.Delete(objectPath, nil, &resp)
	if err != nil {
		return err
	}

	return nil
}

func buildDeletePath(whType sdk.WebHookScope, entityID string, whID string) string {

	var path string

	switch whType {
	case sdk.WebHookScopeSystem:
		path = "hooks/" + whID
	case sdk.WebHookScopeOrg:
		path = sdk.JoinURL("groups", entityID, "hooks", whID)
	case sdk.WebHookScopeRepo:
		path = sdk.JoinURL("projects", entityID, "hooks", whID)
	}

	return path
}
