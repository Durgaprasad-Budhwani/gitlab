package internal

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pinpt/agent.next.gitlab/internal/api"
	"github.com/pinpt/agent.next/sdk"
)

const hookVersion = "1" // change this to upgrade the hook in case the events change

type user struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type webHookRootPayload struct {
	WebHookMainObject json.RawMessage `json:"object_attributes"`
	Project           struct {
		Name string `json:"name"`
		ID   int64  `json:"id"`
	} `json:"project"`
	User         user                       `json:"user"`
	Changes      map[string]json.RawMessage `json:"changes"`
	MergeRequest api.WebhookPullRequest     `json:"merge_request"`
	// push events
	// TotalCommitsCount int64           `json:"total_commits_count"`
	// Commits           []*api.WhCommit `json:"commits"`
}

// WebHook is called when a webhook is received on behalf of the integration
func (i *GitlabIntegration) WebHook(webhook sdk.WebHook) (rerr error) {

	customerID := webhook.CustomerID()
	integrationInstanceID := webhook.IntegrationInstanceID()

	logger := sdk.LogWith(i.logger, "customer_id", customerID)

	pipe := webhook.Pipe()

	event := webhook.Headers()["X-Gitlab-Event"]

	var rootWebHookObject webHookRootPayload
	rerr = json.Unmarshal(webhook.Bytes(), &rootWebHookObject)
	if rerr != nil {
		return
	}

	projectRefID := strconv.FormatInt(rootWebHookObject.Project.ID, 10)

	projectID := sdk.NewSourceCodeRepoID(customerID, projectRefID, gitlabRefType)

	switch event {
	case "Merge Request Hook":

		var pr *api.WebhookPullRequest
		rerr = json.Unmarshal(rootWebHookObject.WebHookMainObject, pr)
		if rerr != nil {
			return
		}

		scPr := pr.ToSourceCodePullRequest(logger, customerID, projectID, gitlabRefType)
		scPr.IntegrationInstanceID = &integrationInstanceID

		rerr = pipe.Write(scPr)
		if rerr != nil {
			return
		}

		pullRequestID := sdk.NewSourceCodePullRequestID(customerID, scPr.RefID, gitlabRefType, projectID)

		ge, err := i.SetQueryConfig(logger, webhook.Config(), i.manager, customerID)
		if err != nil {
			rerr = err
			return
		}

		switch pr.Action {
		case "approved", "unapproved":
			review, err := i.GetReviewFromAction(
				logger,
				ge.qc,
				customerID,
				rootWebHookObject.Project.Name,
				projectID,
				scPr.RefID,
				pullRequestID,
				scPr.RefID,
				pr.IID,
				pr.UpdatedAt,
				rootWebHookObject.User.Username,
				pr.Action)
			if err != nil {
				rerr = err
				return
			}

			review.IntegrationInstanceID = &integrationInstanceID

			rerr = pipe.Write(review)
			if rerr != nil {
				return
			}
		case "update":
			var keyCount int
			var updateKeyExist bool
			for range rootWebHookObject.Changes {
				_, updateKeyExist = rootWebHookObject.Changes["updated_at"]
				keyCount++
				if keyCount > 1 {
					return
				}
			}
			if !updateKeyExist {
				return
			}
			fallthrough
		case "open":
			// TODO: implement commits from push events instead
			var repo *sdk.SourceCodeRepo
			repo.Name = rootWebHookObject.Project.Name
			repo.RefID = projectRefID
			var pr2 *api.PullRequest
			pr2.IID = strconv.FormatInt(pr.IID, 10)
			pr2.RefID = scPr.RefID

			commits, err := ge.FetchPullRequestsCommitsAfter(repo, *pr2, pr.CommonPullRequestFields.UpdatedAt)
			if err != nil {
				return fmt.Errorf("error fetching pull requests commits on webhook, err %d", err)
			}
			for _, c := range commits {
				c.IntegrationInstanceID = &integrationInstanceID
				rerr = pipe.Write(c)
				if rerr != nil {
					return
				}
			}
		}

	case "Push Hook":
		// No need to implement this at this moment
		// After Merge Requeste event it will fetch commits
	case "Note Hook":
		note := api.WebhookNote{}
		rerr = json.Unmarshal(rootWebHookObject.WebHookMainObject, &note)
		if rerr != nil {
			return
		}

		if note.System == false {
			scPr := rootWebHookObject.MergeRequest.ToSourceCodePullRequest(logger, customerID, projectID, gitlabRefType)

			if note.NoteType == "DiffNote" {
				review := note.ToSourceCodePullRequestReview()
				review.CustomerID = customerID
				review.IntegrationInstanceID = sdk.StringPointer(integrationInstanceID)
				review.PullRequestID = scPr.ID
				review.RepoID = projectID

				rerr = pipe.Write(review)
				if rerr != nil {
					return
				}
			} else if note.NoteType == "" {
				prComment := &sdk.SourceCodePullRequestComment{}
				prComment.CustomerID = customerID
				prComment.IntegrationInstanceID = sdk.StringPointer(integrationInstanceID)
				prComment.PullRequestID = scPr.ID

				prComment.RefType = gitlabRefType
				prComment.RefID = strconv.FormatInt(note.ID, 10)
				prComment.URL = note.URL

				sdk.ConvertTimeToDateModel(note.CreatedAt, &prComment.CreatedDate)
				sdk.ConvertTimeToDateModel(note.UpdatedAt, &prComment.UpdatedDate)

				prComment.RepoID = projectID
				prComment.Body = note.Note

				prComment.UserRefID = note.AuthorID

				rerr = pipe.Write(prComment)
				if rerr != nil {
					return
				}

			}
		}
		// SourceCodePullRequestReview
	}

	// TODO: Add webhooks for WORK type

	return
}

func (i *GitlabIntegration) GetReviewFromAction(
	logger sdk.Logger,
	qc api.QueryContext,
	customerID string,
	projectName string,
	projectID string,
	projectRefID string,
	prID string,
	prRefID string,
	prIID int64,
	prUpdatedAt time.Time,
	username string,
	action string) (review *sdk.SourceCodePullRequestReview, rerr error) {

	// TODO: iterate over more notes in rare case it is not foudn in the first 20 notes
	// _, note, err := api.GetGetSinglePullRequestNote(ge.qc, nil, whp.Project.Name, repoRefID, scpr.RefID, wh.IID, whp.User.Username, wh.UpdatedAt, wh.Action)
	_, note, err := api.GetGetSinglePullRequestNote(qc, nil, projectName, projectRefID, prRefID, prIID, username, prUpdatedAt, action)
	if err != nil {
		rerr = err
		return
	}

	review.CustomerID = customerID
	review.RefType = gitlabRefType
	review.RefID = strconv.FormatInt(note.ID, 10)
	review.RepoID = projectID
	review.PullRequestID = prID
	if action == "approved" {
		review.State = sdk.SourceCodePullRequestReviewStateApproved
	} else if action == "unapproved" {
		review.State = sdk.SourceCodePullRequestReviewStateDismissed
	}

	sdk.ConvertTimeToDateModel(note.CreatedAt, &review.CreatedDate)
	review.UserRefID = strconv.FormatInt(note.Author.ID, 10)

	return
}

func (g *GitlabIntegration) registerWebhooks(instance sdk.Instance) (rerr error) {

	// TODO: Add concurrency to webhooks registration

	webhookManager := g.manager.WebHookManager()

	ge, err := g.SetQueryConfig(g.logger, instance.Config(), g.manager, instance.CustomerID())
	if err != nil {
		rerr = err
		return
	}

	user, err := api.LoginUser(ge.qc)
	if err != nil {
		rerr = err
		return
	}

	if !ge.isGitlabCloud && user.IsAdmin {
		rerr = ge.registerSystemWebhook(webhookManager, instance.CustomerID(), instance.IntegrationInstanceID())
		if rerr != nil {
			webhookManager.Errored(instance.CustomerID(), instance.IntegrationInstanceID(), gitlabRefType, "system", sdk.WebHookScopeSystem, err)
			return
		}
	}

	groups, err := api.GroupsAll(ge.qc)
	if err != nil {
		return err
	}
	for _, group := range groups {
		if group.ValidTier {
			user, err := api.GroupUser(ge.qc, group, user.StrID)
			if err != nil {
				group.MarkedToCreateProjectWebHooks = true
				sdk.LogWarn(g.logger, "there was an error trying to get group user access level, will try to create project webhooks instead", "group", group.Name, "user", user.Name, "user_access_level", user.AccessLevel, "err", err)
				return
			}
			if user.AccessLevel >= api.Owner {
				rerr = ge.registerGroupWebhook(webhookManager, instance.CustomerID(), instance.IntegrationInstanceID(), group)
				if rerr != nil {
					group.MarkedToCreateProjectWebHooks = true
					sdk.LogWarn(g.logger, "there was an error trying to create group webhooks, will try to create project webhooks instead", "group", group.Name, "user", user.Name, "user_access_level", user.AccessLevel, "err", err)
					return
				}
			} else {
				group.MarkedToCreateProjectWebHooks = true
				sdk.LogWarn(g.logger, "at least Onwner level access is needed to create webhooks for this group will try to create project webhooks instead", "group", group.Name, "user", user.Name, "user_access_level", user.AccessLevel)
			}
		}
	}

	for _, group := range groups {
		if group.MarkedToCreateProjectWebHooks {
			projects, err := ge.exportGroupRepos(group)
			if err != nil {
				cerr := fmt.Errorf("error trying to get group projects err => %s", err)
				webhookManager.Errored(instance.CustomerID(), instance.IntegrationInstanceID(), gitlabRefType, group.ID, sdk.WebHookScopeOrg, cerr)
				return
			}
			for _, project := range projects {
				user, err := api.ProjectUser(ge.qc, project, user.StrID)
				if err != nil {
					cerr := fmt.Errorf("error trying to get project user user => %s err => %s", user.Name, err)
					webhookManager.Errored(instance.CustomerID(), instance.IntegrationInstanceID(), gitlabRefType, group.ID, sdk.WebHookScopeOrg, cerr)
					return
				}
				if user.AccessLevel >= api.Owner {
					rerr = ge.registerProjectWebhook(webhookManager, instance.CustomerID(), instance.IntegrationInstanceID(), project)
					if rerr != nil {
						cerr := fmt.Errorf("error trying to register project webhooks err => %s", err)
						webhookManager.Errored(instance.CustomerID(), instance.IntegrationInstanceID(), gitlabRefType, group.ID, sdk.WebHookScopeSystem, cerr)
						return
					}
				} else {
					cerr := fmt.Errorf("at least Maintainer level access is needed to create webhooks for this project project => %s user => %s user_access_level %d err => %s", project.Name, user.Name, user.AccessLevel, err)
					webhookManager.Errored(instance.CustomerID(), instance.IntegrationInstanceID(), gitlabRefType, group.ID, sdk.WebHookScopeSystem, cerr)
				}
			}
		}
	}

	// TODO: Refactor registerSystemHook, registerGroupWebHook, registerProjectWebHook

	return
}

func (ge *GitlabExport) registerSystemWebhook(manager sdk.WebHookManager, customerID string, integrationInstanceID string) error {
	if ge.isSystemWebHookInstalled(manager, customerID, integrationInstanceID) {
		return nil
	}

	systeWebHooks, err := ge.getSystemHooks()
	if err != nil {
		return err
	}

	var found bool
	for _, wh := range systeWebHooks {
		if strings.Contains(wh.URL, "event-api") && strings.Contains(wh.URL, "pinpoint.com") && strings.Contains(wh.URL, integrationInstanceID) {
			found = true
			break
		}
	}

	if !found {
		url, err := manager.Create(customerID, integrationInstanceID, gitlabRefType, "system", sdk.WebHookScopeSystem, "scope=system", "version="+hookVersion)
		if err != nil {
			manager.Delete(customerID, integrationInstanceID, gitlabRefType, "system", sdk.WebHookScopeSystem)
			return err
		}
		err = api.CreateSystemWebHook(ge.qc, url)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ge *GitlabExport) isSystemWebHookInstalled(manager sdk.WebHookManager, customerID string, integrationInstanceID string) bool {
	// TODO: define system hook scope type in agent.next sdk
	if manager.Exists(customerID, integrationInstanceID, gitlabRefType, "system", sdk.WebHookScopeSystem) {
		theurl, _ := manager.HookURL(customerID, integrationInstanceID, gitlabRefType, "system", sdk.WebHookScopeSystem)
		// check and see if we need to upgrade our hook
		if !strings.Contains(theurl, "version="+hookVersion) {
			manager.Delete(customerID, integrationInstanceID, gitlabRefType, "system", sdk.WebHookScopeSystem)
			return false
		}
		return true
	}
	return false
}

func (ge *GitlabExport) getSystemHooks() (gwhs []*api.GitlabWebhook, rerr error) {
	rerr = api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (np api.NextPage, rerr error) {
		pi, whs, err := api.GetSystemWebHookPage(ge.qc, params)
		if err != nil {
			return pi, err
		}
		gwhs = append(gwhs, whs...)
		return
	})
	return
}

func (ge *GitlabExport) registerGroupWebhook(manager sdk.WebHookManager, customerID string, integrationInstanceID string, group *api.Group) error {
	if ge.isGroupWebHookInstalled(manager, customerID, integrationInstanceID, group) {
		return nil
	}

	groupWebHooks, err := ge.getGroupHooks(group)
	if err != nil {
		return err
	}

	var found bool
	for _, wh := range groupWebHooks {
		if strings.Contains(wh.URL, "event-api") && strings.Contains(wh.URL, "pinpoint.com") && strings.Contains(wh.URL, integrationInstanceID) {
			found = true
			break
		}
	}

	if !found {
		url, err := manager.Create(customerID, integrationInstanceID, gitlabRefType, group.ID, sdk.WebHookScopeOrg, "scope=org", "version="+hookVersion)
		if err != nil {
			manager.Delete(customerID, integrationInstanceID, gitlabRefType, group.ID, sdk.WebHookScopeOrg)
			return err
		}
		err = api.CreateGroupWebHook(ge.qc, group, url)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ge *GitlabExport) isGroupWebHookInstalled(manager sdk.WebHookManager, customerID string, integrationInstanceID string, group *api.Group) bool {
	// TODO: define system hook scope type in agent.next sdk
	if manager.Exists(customerID, integrationInstanceID, gitlabRefType, group.ID, sdk.WebHookScopeOrg) {
		theurl, _ := manager.HookURL(customerID, integrationInstanceID, gitlabRefType, group.ID, sdk.WebHookScopeOrg)
		// check and see if we need to upgrade our hook
		if !strings.Contains(theurl, "version="+hookVersion) {
			manager.Delete(customerID, integrationInstanceID, gitlabRefType, group.ID, sdk.WebHookScopeOrg)
			return false
		}
		return true
	}
	return false
}

func (ge *GitlabExport) getGroupHooks(group *api.Group) (gwhs []*api.GitlabWebhook, rerr error) {
	rerr = api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (np api.NextPage, rerr error) {
		pi, whs, err := api.GetGroupWebHookPage(ge.qc, group, params)
		if err != nil {
			return pi, err
		}
		gwhs = append(gwhs, whs...)
		return
	})
	return
}

func (ge *GitlabExport) registerProjectWebhook(manager sdk.WebHookManager, customerID string, integrationInstanceID string, project *sdk.SourceCodeRepo) error {
	if ge.isProjectWebHookInstalled(manager, customerID, integrationInstanceID, project) {
		return nil
	}

	projectWebHooks, err := ge.getProjectHooks(project)
	if err != nil {
		return err
	}

	var found bool
	for _, wh := range projectWebHooks {
		if strings.Contains(wh.URL, "event-api") && strings.Contains(wh.URL, "pinpoint.com") && strings.Contains(wh.URL, integrationInstanceID) {
			found = true
			break
		}
	}

	if !found {
		url, err := manager.Create(customerID, integrationInstanceID, gitlabRefType, project.RefID, sdk.WebHookScopeRepo, "scope=repo", "version="+hookVersion)
		if err != nil {
			manager.Delete(customerID, integrationInstanceID, gitlabRefType, project.RefID, sdk.WebHookScopeRepo)
			return err
		}
		err = api.CreateProjectWebHook(ge.qc, project, url)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ge *GitlabExport) isProjectWebHookInstalled(manager sdk.WebHookManager, customerID string, integrationInstanceID string, project *sdk.SourceCodeRepo) bool {
	// TODO: define system hook scope type in agent.next sdk
	if manager.Exists(customerID, integrationInstanceID, gitlabRefType, project.RefID, sdk.WebHookScopeRepo) {
		theurl, _ := manager.HookURL(customerID, integrationInstanceID, gitlabRefType, project.RefID, sdk.WebHookScopeRepo)
		// check and see if we need to upgrade our hook
		if !strings.Contains(theurl, "version="+hookVersion) {
			manager.Delete(customerID, integrationInstanceID, gitlabRefType, project.RefID, sdk.WebHookScopeRepo)
			return false
		}
		return true
	}
	return false
}

func (ge *GitlabExport) getProjectHooks(project *sdk.SourceCodeRepo) (gwhs []*api.GitlabWebhook, rerr error) {
	rerr = api.Paginate(ge.logger, "", time.Time{}, func(log sdk.Logger, params url.Values, t time.Time) (np api.NextPage, rerr error) {
		pi, whs, err := api.GetProjectWebHookPage(ge.qc, project, params)
		if err != nil {
			return pi, err
		}
		gwhs = append(gwhs, whs...)
		return
	})
	return
}
