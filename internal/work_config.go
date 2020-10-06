package internal

import (
	"strings"

	"github.com/pinpt/agent/v4/sdk"
	"github.com/pinpt/gitlab/internal/api"
)

type labelMap struct {
	Mapped sdk.WorkIssueTypeMappedType
}

func (i *GitlabExport) workConfig() error {

	labels := map[string]labelMap{
		api.BugIssueType: {
			sdk.WorkIssueTypeMappedTypeBug,
		},
		api.EpicIssueType: {
			sdk.WorkIssueTypeMappedTypeEpic,
		},
		api.EnhancementIssueType: {
			sdk.WorkIssueTypeMappedTypeEnhancement,
		},
	}

	wc := &sdk.WorkConfig{}
	wc.ID = sdk.NewWorkConfigID(i.qc.CustomerID, i.qc.RefType, *i.integrationInstanceID)
	wc.CreatedAt = sdk.EpochNow()
	wc.UpdatedAt = sdk.EpochNow()
	wc.CustomerID = i.qc.CustomerID
	wc.IntegrationInstanceID = *i.integrationInstanceID
	wc.RefType = i.qc.RefType
	wc.Statuses = sdk.WorkConfigStatuses{
		OpenStatus:   []string{api.OpenedState, strings.ToLower(api.OpenedState)},
		ClosedStatus: []string{api.ClosedState, strings.ToLower(api.ClosedState)},
	}

	if err := i.pipe.Write(wc); err != nil {
		return err
	}

	for key, lbl := range labels {
		issuetype := &sdk.WorkIssueType{}
		issuetype.CustomerID = i.qc.CustomerID
		issuetype.RefID = key
		issuetype.RefType = i.qc.RefType
		issuetype.Name = key
		issuetype.IntegrationInstanceID = sdk.StringPointer(i.integrationInstanceID)
		issuetype.Description = sdk.StringPointer(key)
		// issuetype.IconURL NA
		issuetype.MappedType = lbl.Mapped
		issuetype.ID = sdk.NewWorkIssueTypeID(i.qc.CustomerID, i.qc.RefType, key)
		if err := i.pipe.Write(issuetype); err != nil {
			return err
		}
	}

	return nil
}
