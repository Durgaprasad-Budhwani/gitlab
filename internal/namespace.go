package internal

import (
	"encoding/json"
	"fmt"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

// NamespaceManager namespace manager
type NamespaceManager struct {
	state  sdk.State
	logger sdk.Logger
}

const namespaceStatecacheKey = "namespace_"

// SaveNamespace save namespace into state
func (u *NamespaceManager) SaveNamespace(namespace api.Namespace) error {
	cachekey := namespaceStatecacheKey + namespace.ID

	bts, err := json.Marshal(namespace)
	if err != nil {
		return err
	}

	return u.state.Set(cachekey, string(bts))
}

// GetNamespace get namespace custom data
func (u *NamespaceManager) GetNamespace(namespaceID string) (ns api.Namespace, err error) {
	cachekey := namespaceStatecacheKey + namespaceID

	ok, err := u.state.Get(cachekey, &ns)
	if err != nil {
		return
	}

	if !ok {
		sdk.LogDebug(u.logger, "key not found", "key", cachekey)
		return ns, fmt.Errorf("key not found, key = %s", cachekey)
	}

	return
}

// NewNamespaceManager returns a new instance
func NewNamespaceManager(logger sdk.Logger, state sdk.State) *NamespaceManager {
	return &NamespaceManager{
		state:  state,
		logger: logger,
	}
}
