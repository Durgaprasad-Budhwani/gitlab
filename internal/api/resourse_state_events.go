package api

import (
	"net/url"
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

func getOpenCloseIssueHistoryPage(
	qc QueryContext,
	project *sdk.SourceCodeRepo,
	issueIID string,
	stopOnUpdatedAt time.Time,
	params url.Values) (pi NextPage, rse []*ResourceStateEvents, err error) {

	sdk.LogDebug(qc.Logger, "work issue resource_state_events", "project", project.RefID)

	objectPath := sdk.JoinURL("projects", url.QueryEscape(project.RefID), "issues", issueIID, "resource_state_events")

	pi, err = qc.Get(objectPath, nil, &rse)
	if err != nil {
		return
	}

	return
}

// GetOpenClosedIssueHistory get open closed issue history
func GetOpenClosedIssueHistory(
	qc QueryContext,
	project *sdk.SourceCodeRepo,
	issueIID string) (rse []*ResourceStateEvents, err error) {

	err = Paginate(qc.Logger, "", time.Time{}, func(log sdk.Logger, params url.Values, stopOnUpdatedAt time.Time) (NextPage, error) {
		np, arr, err := getOpenCloseIssueHistoryPage(qc, project, issueIID, stopOnUpdatedAt, params)
		if err != nil {
			return np, err
		}
		rse = append(rse, arr...)

		return np, nil
	})

	return
}
