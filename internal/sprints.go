package internal

import "github.com/pinpt/agent/v4/sdk"

func (ge *GitlabExport) exportSprints(sprints []*sdk.AgileSprint) error {
	for _, sprint := range sprints {
		sprint.IntegrationInstanceID = ge.integrationInstanceID
		ge.qc.WorkManager.SetSprintColumnsIssuesProjectIDs(sprint)
		if err := ge.pipe.Write(sprint); err != nil {
			return err
		}
	}

	return nil
}
