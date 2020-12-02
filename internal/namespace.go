package internal

import (
	"strconv"
	"strings"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

// Namespace internal namespace
type Namespace struct {
	ID                            string
	Name                          string
	Path                          string
	FullPath                      string
	ValidTier                     bool
	MarkedToCreateProjectWebHooks bool
	Visibility                    string
	AvatarURL                     string
	Kind                          string
}

func (e *GitlabExport2) getSelectedNamespacesIfAny(logger sdk.Logger, namespaces chan *Namespace) error {

	allNamespaces, err := api.AllNamespaces2(e.qc, logger)
	if err != nil {
		return err
	}

	if e.config.Accounts == nil {
		for _, namespace := range allNamespaces {
			if namespace.ParentID != 0 { // skip subgroups
				continue
			}

			if !strings.Contains(namespace.AvatarURL, "https") && namespace.AvatarURL != "" {
				namespace.AvatarURL = e.qc.BaseURL() + namespace.AvatarURL
			}

			namespaces <- toNamespace(namespace)
		}
	} else {
		accounts := *e.config.Accounts
		for _, namespace := range allNamespaces {
			account, ok := accounts[strconv.FormatInt( namespace.ID,10)]
			if !ok {
				continue
			}

			if account.Selected != nil && *account.Selected {
				namespaces <- toNamespace(namespace)
			}
		}
	}

	close(namespaces)

	return nil
}


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

func toNamespace(namespace *api.GitlabNamespace) *Namespace {
	return &Namespace{
		ID:        strconv.FormatInt(namespace.ID, 10),
		Name:      namespace.Name,
		FullPath:  namespace.FullPath,
		ValidTier: namespace.MembersCountWithDescendants != nil,
		AvatarURL: namespace.AvatarURL,
		Kind:      namespace.Kind,
		Path:      namespace.Path,
	}
}
