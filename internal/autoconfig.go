package internal

import (
	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (g *GitlabIntegration) AutoConfigure(autoconfig sdk.AutoConfigure) (*sdk.Config, error) {
	config := autoconfig.Config()
	logger := g.logger

	ge, err := g.SetQueryConfig(logger, config, g.manager, autoconfig.CustomerID())
	if err != nil {
		return nil, err
	}

	viewer, err := api.LoginUser(ge.qc)
	if err != nil {
		return nil, err
	}
	var acct sdk.ConfigAccount
	acct.ID = viewer.StrID
	acct.Public = false
	acct.Type = sdk.ConfigAccountTypeUser
	accounts := make(sdk.ConfigAccounts)
	accounts[acct.ID] = &acct
	config.Accounts = &accounts
	return &config, nil
}
