package internal

import "github.com/pinpt/agent.next/sdk"

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
