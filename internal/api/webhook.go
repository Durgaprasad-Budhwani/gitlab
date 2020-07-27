package api

import (
	"net/url"

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

func CreateSystemWebHook(qc QueryContext, eventAPIWebhookURL string) error {

	sdk.LogDebug(qc.Logger, "system webhooks")

	objectPath := sdk.JoinURL("hooks")

	params := url.Values{}
	params.Set("url", eventAPIWebhookURL)
	params.Set("token", "token")
	// groups/projects webhooks will handle merge_requests_events
	// params.Set("merge_requests_events", "true")
	// params.Set("push_events", "true")
	params.Set("repository_update_events", "true")
	params.Set("enable_ssl_verification", "true")

	var resp interface{}

	_, err := qc.Post(objectPath, params, nil, &resp)
	if err != nil {
		return err
	}

	return nil
}

func CreateGroupWebHook(qc QueryContext, group *Group, eventAPIWebhookURL string) error {

	sdk.LogDebug(qc.Logger, "group webhooks", "group_name", group.Name, "group_id", group.ID)

	objectPath := sdk.JoinURL("groups", group.ID, "hooks")

	params := url.Values{}
	params.Set("url", eventAPIWebhookURL)
	params.Set("push_events", "true")
	params.Set("merge_requests_events", "true")
	params.Set("note_events", "true")
	params.Set("enable_ssl_verification", "true")

	var resp interface{}

	_, err := qc.Post(objectPath, params, nil, &resp)
	if err != nil {
		return err
	}

	return nil
}

func CreateProjectWebHook(qc QueryContext, project *sdk.SourceCodeRepo, eventAPIWebhookURL string) error {

	sdk.LogDebug(qc.Logger, "group webhooks", "project_name", project.Name, "project_id", project.RefID)

	objectPath := sdk.JoinURL("projects", project.RefID, "hooks")

	params := url.Values{}
	params.Set("url", eventAPIWebhookURL)
	params.Set("push_events", "true")
	params.Set("merge_requests_events", "true")
	params.Set("note_events", "true")
	params.Set("enable_ssl_verification", "true")

	var resp interface{}

	_, err := qc.Post(objectPath, params, nil, &resp)
	if err != nil {
		return err
	}

	return nil
}

type GitlabWebhook struct {
	URL string `json:"url"`
}

func GetSystemWebHookPage(qc QueryContext, params url.Values) (page NextPage, gwhs []*GitlabWebhook, err error) {

	sdk.LogDebug(qc.Logger, "system webhooks")

	objectPath := sdk.JoinURL("hooks")

	page, err = qc.Get(objectPath, params, &gwhs)

	return
}

func GetGroupWebHookPage(qc QueryContext, group *Group, params url.Values) (page NextPage, gwhs []*GitlabWebhook, err error) {

	sdk.LogDebug(qc.Logger, "group webhooks", "group_id", group.ID, "group_name", group.Name, "params", params)

	objectPath := sdk.JoinURL("groups", group.ID, "hooks")

	page, err = qc.Get(objectPath, params, &gwhs)

	return
}

func GetProjectWebHookPage(qc QueryContext, project *sdk.SourceCodeRepo, params url.Values) (page NextPage, gwhs []*GitlabWebhook, err error) {

	sdk.LogDebug(qc.Logger, "project webhooks", "repo_id", project.RefID, "repo_name", project.Name, "params", params)

	objectPath := sdk.JoinURL("projects", project.RefID, "hooks")

	page, err = qc.Get(objectPath, params, &gwhs)

	return
}
