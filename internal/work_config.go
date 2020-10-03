package internal

import (
	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

func (i *GitlabExport) workConfig() error {

	wc := &sdk.WorkConfig{}
	wc.ID = sdk.NewWorkConfigID(i.qc.CustomerID, i.qc.RefType, *i.integrationInstanceID)
	wc.CreatedAt = sdk.EpochNow()
	wc.UpdatedAt = sdk.EpochNow()
	wc.CustomerID = i.qc.CustomerID
	wc.IntegrationInstanceID = *i.integrationInstanceID
	wc.RefType = i.qc.RefType
	wc.Statuses = sdk.WorkConfigStatuses{
		OpenStatus:   []string{api.OpenedState},
		ClosedStatus: []string{api.ClosedState},
	}

	return i.pipe.Write(wc)
}
