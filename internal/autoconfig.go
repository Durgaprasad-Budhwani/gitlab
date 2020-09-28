package internal

import (
	"github.com/pinpt/gitlab/internal/api"
	"github.com/pinpt/agent/v4/sdk"
)

func (g *GitlabIntegration) AutoConfigure(autoconfig sdk.AutoConfigure) (*sdk.Config, error) {
	config := autoconfig.Config()

	logger := sdk.LogWith(g.logger, "event", "autoconfigure", "customer_id", autoconfig.CustomerID(), "integration_instance_id", autoconfig.IntegrationInstanceID())

	sdk.LogInfo(logger, "auto-configure started")

	ge, err := g.SetQueryConfig(logger, config, g.manager, autoconfig.CustomerID())
	if err != nil {
		return nil, err
	}

	namespaces, err := api.AllNamespaces(ge.qc)
	if err != nil {
		return nil, err
	}

	accounts := make(sdk.ConfigAccounts)
	if config.Scope != nil && *config.Scope == sdk.OrgScope {
		for _, ns := range namespaces {
			if ns.Kind == "group" {
				var repos []*sdk.SourceCodeRepo
				err = ge.fetchNamespaceProjectsRepos(ns, func(repo *sdk.SourceCodeRepo) {
					repos = append(repos, repo)
				})
				if err != nil {
					return nil, err
				}

				acct := &sdk.ConfigAccount{}
				acct.ID = ns.ID
				acct.Type = sdk.ConfigAccountTypeOrg
				acct.Selected = sdk.BoolPointer(true)
				reposCount := int64(len(repos))
				acct.TotalCount = &reposCount
				acct.AvatarURL = sdk.StringPointer(ns.AvatarURL)
				accounts[acct.ID] = acct
			}
		}
	} else {
		viewer, err := api.LoginUser(ge.qc)
		if err != nil {
			return nil, err
		}
		for _, ns := range namespaces {
			if ns.Kind == "user" && viewer.Username == ns.Path {
				var repos []*sdk.SourceCodeRepo
				err = ge.fetchNamespaceProjectsRepos(ns, func(repo *sdk.SourceCodeRepo) {
					repos = append(repos, repo)
				})
				if err != nil {
					return nil, err
				}
				acct := &sdk.ConfigAccount{}
				acct.ID = ns.ID
				acct.Type = sdk.ConfigAccountTypeUser
				acct.Selected = sdk.BoolPointer(true)
				reposCount := int64(len(repos))
				acct.TotalCount = &reposCount
				acct.AvatarURL = sdk.StringPointer(ns.AvatarURL)
				accounts[acct.ID] = acct
			}
		}
	}

	config.Accounts = &accounts

	return &config, nil
}
