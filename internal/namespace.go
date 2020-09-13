package internal

import (
	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func getNamespacesSelectedAccounts(qc api.QueryContext, accounts *sdk.ConfigAccounts) ([]*api.Namespace, error) {

	filteredNamespaces := make([]*api.Namespace, 0)

	allNamespaces, err := api.AllNamespaces(qc)
	if err != nil {
		return nil, err
	}

	for _, namespace := range allNamespaces {
		r := *accounts
		nsSelected := r[namespace.ID].Selected
		if nsSelected != nil && *nsSelected {
			filteredNamespaces = append(filteredNamespaces, namespace)
		}
	}

	return filteredNamespaces, nil
}
