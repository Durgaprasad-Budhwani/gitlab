package internal

import "github.com/pinpt/agent.next/sdk"

// RepoProjectManager repo/project manager
type RepoProjectManager struct {
	logger              sdk.Logger
	state               sdk.State
	pipe                sdk.Pipe
	curretReposExported []*sdk.SourceCodeRepo
}

const reposProjectsProcessedKey = "repos_projects_processed"

type stateRepos map[string]*sdk.SourceCodeRepo

// PersistRepos persist repos
func (r *RepoProjectManager) PersistRepos() error {

	stateRepos := make(stateRepos)

	_, err := r.state.Get(reposProjectsProcessedKey, &stateRepos)
	if err != nil {
		return err
	}

	deativateRepos := make(map[string]bool)
	for _, crepo := range stateRepos {
		deativateRepos[crepo.ID] = true
	}

	for _, repo := range r.curretReposExported {
		stateRepos[repo.ID] = repo
		deativateRepos[repo.ID] = false
	}

	for _, repo := range stateRepos {
		if deativateRepos[repo.ID] {
			delete(stateRepos, repo.ID)
			sdk.LogDebug(r.logger, "deactivating repo", "repo", repo)
			if err := r.deactivateRepoAndProject(repo); err != nil {
				return err
			}
		}
	}

	err = r.state.Set(reposProjectsProcessedKey, stateRepos)
	if err != nil {
		return err
	}

	return nil

}

func (r *RepoProjectManager) deactivateRepoAndProject(repo *sdk.SourceCodeRepo) error {

	repo.Active = false
	repo.UpdatedAt = sdk.EpochNow()
	if err := r.pipe.Write(repo); err != nil {
		return err
	}
	project := ToProject(repo)
	project.Active = false
	project.UpdatedAt = sdk.EpochNow()
	if err := r.pipe.Write(project); err != nil {
		return err
	}

	return nil
}

// AddRepo add repo
func (r *RepoProjectManager) AddRepo(repo *sdk.SourceCodeRepo) {
	r.curretReposExported = append(r.curretReposExported, repo)
}

// DeactivateReposAndProjects deactivate repos and projects
func (r *RepoProjectManager) DeactivateReposAndProjects() error {

	stateRepos := make(stateRepos)

	_, err := r.state.Get(reposProjectsProcessedKey, &stateRepos)
	if err != nil {
		return err
	}

	for _, repo := range stateRepos {
		err = r.deactivateRepoAndProject(repo)
		if err != nil {
			return err
		}
	}

	return r.state.Delete(reposProjectsProcessedKey)

}

// NewRepoProjectManager new repo project manager
func NewRepoProjectManager(logger sdk.Logger, state sdk.State, pipe sdk.Pipe) *RepoProjectManager {
	return &RepoProjectManager{
		logger:              logger,
		state:               state,
		pipe:                pipe,
		curretReposExported: make([]*sdk.SourceCodeRepo, 0),
	}
}
