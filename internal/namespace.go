package internal

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
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

// Namespace internal namespace
type Namespace struct {
	ID                            string
	Name                          string
	Path                          string `json:"path"`
	FullPath                      string
	ValidTier                     bool `json:"valid_tier"`
	MarkedToCreateProjectWebHooks bool
	Visibility                    string
	AvatarURL                     string
	Kind                          string
}

func (g *GitlabExport2) getAllNamespaces(logger sdk.Logger, config sdk.Config) ([]*Namespace, error) {
	allNamespaces := make([]*Namespace, 0)
	if config.Accounts == nil {
		namespaces, err := api.AllNamespaces2(g.qc, logger)
		if err != nil {
			return nil, fmt.Errorf("error getting all namespaces %s", err)
		}

		for _, namespace := range namespaces {
			// skip subgroups
			if namespace.ParentID != 0 {
				continue
			}

			if !strings.Contains(namespace.AvatarURL, "https") && namespace.AvatarURL != "" {
				namespace.AvatarURL = g.baseURL + namespace.AvatarURL
			}

			n := &Namespace{
				ID:        strconv.FormatInt(namespace.ID, 10),
				Name:      namespace.Name,
				FullPath:  namespace.FullPath,
				ValidTier: namespace.MembersCountWithDescendants != nil,
				AvatarURL: namespace.AvatarURL,
				Kind:      namespace.Kind,
				Path:      namespace.Path,
			}

			allNamespaces = append(allNamespaces, n)
		}

	}
	//else {
	//	namespaces, err := getNamespacesSelectedAccounts(gexport.qc, config.Accounts)
	//	if err != nil {
	//		sdk.LogError(logger, "error getting data accounts", "err", err)
	//		return nil, fmt.Errorf("error getting selected namespaces %s", err)
	//	}
	//	allNamespaces = append(allNamespaces, namespaces...)
	//}

	return nil, nil
}
