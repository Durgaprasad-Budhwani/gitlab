package internal

import "github.com/pinpt/agent.next/sdk"

func (ge *GitlabExport) exportSprints(sprints []*sdk.AgileSprint) error {
	for _, sprint := range sprints {
		_ = sprint
		// sprint.BoardIds =
		// sprint.Columns =
		// sprint.ProjectIds =
		// sprint.IssueIds =
	}

	return nil
}
