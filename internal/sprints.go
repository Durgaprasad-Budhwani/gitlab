package internal

import "github.com/pinpt/agent.next/sdk"

func (ge *GitlabExport) exportReposSprints(repos []*sdk.SourceCodeRepo) error {
	for _, repo := range repos {
		if err := ge.exportRepoSprints(repo); err != nil {
			return err
		}
	}

	return nil
}
