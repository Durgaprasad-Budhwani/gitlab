package internal

import (
	"net/url"
	"strings"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (ge *GitlabExport) isWebHookInstalled(webhookType sdk.WebHookScope, manager sdk.WebHookManager, customerID, integrationInstanceID, entityID string) bool {
	if manager.Exists(customerID, integrationInstanceID, gitlabRefType, entityID, sdk.WebHookScopeSystem) {
		theurl, _ := manager.HookURL(customerID, integrationInstanceID, gitlabRefType, entityID, sdk.WebHookScopeSystem)
		// check and see if we need to upgrade our hook
		if !strings.Contains(theurl, "version="+hookVersion) {
			manager.Delete(customerID, integrationInstanceID, gitlabRefType, entityID, sdk.WebHookScopeSystem)
			return false
		}
		return true
	}
	return false
}

func (ge *GitlabExport) getHooks(webhookType sdk.WebHookScope, entityID, entityName string) (gwhs []*api.GitlabWebhook, rerr error) {
	rerr = api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (np api.NextPage, rerr error) {
		pi, whs, err := api.GetWebHooksPage(webhookType, ge.qc, entityID, entityName, params)
		if err != nil {
			return pi, err
		}
		gwhs = append(gwhs, whs...)
		return
	})
	return
}
