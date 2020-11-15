package internal

import (
	"reflect"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

// Mutation is called when a mutation is received on behalf of the integration
func (g *GitlabIntegration) Mutation(mutation sdk.Mutation) (*sdk.MutationResponse, error) {

	logger := sdk.LogWith(mutation.Logger(), "integration_event", "mutation")

	sdk.LogInfo(logger, "mutation request received", "action", mutation.Action(), "id", mutation.ID(), "model", mutation.Model())



	user := mutation.User()
	var c sdk.Config
	c.APIKeyAuth = user.APIKeyAuth
	c.BasicAuth = user.BasicAuth
	c.OAuth2Auth = user.OAuth2Auth

	ge, err := g.SetQueryConfig(logger, c, g.manager, mutation.CustomerID())
	if err != nil {
		return nil, err
	}
	ge.qc.Pipe = mutation.Pipe()
	ge.qc.State = mutation.State()
	ge.qc.WorkManager = NewWorkManager(logger, mutation.State())
	ge.qc.IntegrationInstanceID = mutation.IntegrationInstanceID()

	sdk.LogInfo(logger, "recovering work manager state")
	if err := ge.qc.WorkManager.Restore(); err != nil {
		sdk.LogError(logger, "error recovering work manager state", "err", err)
		return nil, err
	}

	switch mutationModelType := mutation.Payload().(type) {
	// Issue
	case *sdk.WorkIssueUpdateMutation:
		return api.UpdateIssueFromMutation(ge.qc, mutation, mutationModelType)
	case *sdk.WorkIssueCreateMutation:
		return ge.createIssue(mutationModelType)
		// switch *mutationModelType.Type.Name {
		// case api.BugIssueType:
		// 	return api.CreateWorkIssue(ge.qc, mutationModelType, "")
		// case api.IncidentIssueType:
		// 	return api.CreateWorkIssue(ge.qc, mutationModelType, api.IncidentIssueType)
		// case api.EnhancementIssueType:
		// 	return api.CreateWorkIssue(ge.qc, mutationModelType, api.EnhancementIssueType)
		// case api.EpicIssueType:
		// 	return api.CreateEpic(ge.qc, mutationModelType)
		// }

	// Sprint
	case *sdk.AgileSprintUpdateMutation:
		sdk.LogInfo(logger, "not action for this mutation type")
		// Uncomment when Group Name/ID is sent by the UI
		// return api.UpdateSprint(ge.qc, mutation, mutationModelType)
	case *sdk.AgileSprintCreateMutation:
		sdk.LogInfo(logger, "not action for this mutation type")
		// Uncomment when Group Name/ID is sent by the UI
		// return api.CreateSprintFromMutation(ge.qc, mutation.ID(), mutationModelType)
	}
	sdk.LogInfo(logger, "unhandled mutation request", "type", reflect.TypeOf(mutation.Payload()))
	return nil, nil
}
