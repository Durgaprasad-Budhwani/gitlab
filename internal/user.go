package internal

import (
	"net/url"
	"sync"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

func (ge *GitlabExport) exportRepoUsers(repo *sdk.SourceCodeRepo) error {

	return ge.exportUsers(repo, func(user *sdk.SourceCodeUser) error {
		user.IntegrationInstanceID = ge.integrationInstanceID
		return ge.pipe.Write(user)
	})

}

func (ge *GitlabExport) exportProjectUsers(project *sdk.WorkProject) (usermap api.UsernameMap, rerr error) {

	usermap = make(api.UsernameMap)

	rerr = ge.exportUsers(ToRepo(project), func(user *sdk.SourceCodeUser) error {
		usermap[*user.Username] = user.RefID
		user.IntegrationInstanceID = ge.integrationInstanceID
		return ge.pipe.Write(toWorkUser(user))
	})

	return
}

func toWorkUser(user *sdk.SourceCodeUser) *sdk.WorkUser {
	var username string
	if user.Username != nil {
		username = *user.Username
	}
	return &sdk.WorkUser{
		AssociatedRefID: user.AssociatedRefID,
		AvatarURL:       user.AvatarURL,
		CustomerID:      user.CustomerID,
		Email:           user.Email,
		ID:              user.ID,
		Name:            user.Name,
		RefID:           user.RefID,
		RefType:         user.RefType,
		URL:             user.URL,
		Username:        username,
		Hashcode:        user.Hashcode,
	}
}

type callBackSourceUser func(item *sdk.SourceCodeUser) error

func (ge *GitlabExport) exportUsers(repo *sdk.SourceCodeRepo, callback callBackSourceUser) (rerr error) {
	return api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, err error) {
		pi, arr, err := api.RepoUsersPage(ge.qc, repo, params)
		if err != nil {
			return
		}
		for _, item := range arr {
			if err = callback(item); err != nil {
				return
			}
		}
		return
	})
}

func (ge *GitlabExport) exportEnterpriseUsers() error {
	return ge.fetchEnterpriseUsers(func(user *sdk.SourceCodeUser) error {
		user.IntegrationInstanceID = ge.integrationInstanceID
		return ge.pipe.Write(user)
	})
}

func (ge *GitlabExport) fetchEnterpriseUsers(callback callBackSourceUser) (rerr error) {
	return api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (pi api.NextPage, err error) {
		params.Set("membership", "true")
		pi, arr, err := api.UsersPage(ge.qc, params)
		if err != nil {
			return
		}
		for _, item := range arr {
			if err = callback(item); err != nil {
				return
			}
		}
		return
	})
}

// UserManager is a manager for users
// easyjson:skip
type UserManager struct {
	customerID  string
	users       map[string]bool
	control     sdk.Control
	pipe        sdk.Pipe
	integration *GitlabIntegration
	mu          sync.Mutex
	instanceid  string
	state       sdk.State
	historical  bool
}

const userStatecacheKey = "user_"

func (u *UserManager) EmitGitUser(logger sdk.Logger, author api.GitUser2) error {
	refID := author.RefID(u.customerID)
	if refID == "" {
		return nil
	}
	// hold lock while we determine if this is a user we need to lookup
	u.mu.Lock()
	if u.users[refID] {
		u.mu.Unlock()
		return nil
	}
	user := author.ToModel(u.customerID, u.instanceid)
	hash := user.Hash()
	cachekey := userStatecacheKey + refID
	u.users[refID] = true
	if !u.historical {
		var cacheValue string
		found, _ := u.state.Get(cachekey, &cacheValue)
		if found && cacheValue == hash {
			// already cached with the same hashcode so we can skip emit
			u.mu.Unlock()
			return nil
		}
	}
	u.mu.Unlock()
	if err := u.pipe.Write(user); err != nil {
		return err
	}
	return u.state.SetWithExpires(cachekey, hash, time.Hour)
}

// NewUserManager returns a new instance
func NewUserManager(customerID string, control sdk.Control, state sdk.State, pipe sdk.Pipe, instanceid string) *UserManager {
	return &UserManager{
		customerID: customerID,
		users:      make(map[string]bool),
		control:    control,
		pipe:       pipe,
		state:      state,
		instanceid: instanceid,
	}
}
