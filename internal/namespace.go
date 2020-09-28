package internal

import (
	"github.com/pinpt/gitlab/internal/api"
	"github.com/pinpt/agent/v4/sdk"
)

func getNamespacesSelectedAccounts(qc api.QueryContext, accounts *sdk.ConfigAccounts) ([]*api.Namespace, error) {

	filteredNamespaces := make([]*api.Namespace, 0)

	allNamespaces, err := api.AllNamespaces(qc)
	if err != nil {
		return nil, err
	}

	for _, namespace := range allNamespaces {
		r := *accounts
		account, ok := r[namespace.ID]
		if !ok {
			continue
		}

		if account.Selected != nil && *account.Selected {
			filteredNamespaces = append(filteredNamespaces, namespace)
		}
	}

	return filteredNamespaces, nil
}
