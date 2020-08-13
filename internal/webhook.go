package internal

import (
	"encoding/json"
	"fmt"
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
	EventName    string                     `json:"event_name"`
	ProjectID    string                     `json:"project_id"`
	UserID       string                     `json:"user_id"`
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
	case "System Hook":
		switch rootWebHookObject.EventName {
		case "repository_update", "project_update", "project_rename":
			ge, err := i.SetQueryConfig(logger, webhook.Config(), i.manager, customerID)
			if err != nil {
				rerr = err
				return
			}
			repo, err := api.ProjectByID(ge.qc, rootWebHookObject.ProjectID)
			if err != nil {
				rerr = err
				return
			}
			repo.IntegrationInstanceID = &integrationInstanceID
			rerr = pipe.Write(repo)
			if rerr != nil {
				return
			}
		case "user_create", "user_rename":
			ge, err := i.SetQueryConfig(logger, webhook.Config(), i.manager, customerID)
			if err != nil {
				rerr = err
				return
			}

			user, err := api.UserByID(ge.qc, rootWebHookObject.ProjectID)
			if err != nil {
				rerr = err
				return
			}
			user.IntegrationInstanceID = &integrationInstanceID
			rerr = pipe.Write(user)
			if rerr != nil {
				return
			}
			// TODO: check if these are useful
			// case "user_add_to_team":
			// case "user_update_for_team":
			// case "user_add_to_group":
			// case "user_update_for_group":
			// case "group_create":
			// case "user_remove_from_team":
			// case "user_destroy":
			// case "group_destroy":
			// case "group_rename":
			// case "user_remove_from_group":
		}
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

func (g *GitlabIntegration) registerWebhooks(ge GitlabExport) error {

	// TODO: Add concurrency to webhooks registration
	customerID := ge.qc.CustomerID
	integrationInstanceID := ge.integrationInstanceID
	webhookManager := g.manager.WebHookManager()

	wr := webHookRegistration{
		customerID:            customerID,
		integrationInstanceID: *integrationInstanceID,
		manager:               webhookManager,
		ge:                    &ge,
	}

	user, err := api.LoginUser(ge.qc)
	if err != nil {
		return err
	}

	if !ge.isGitlabCloud && user.IsAdmin {
		err = wr.registerWebhook(sdk.WebHookScopeSystem, "", "")
		if err != nil {
			sdk.LogDebug(ge.logger, "error registering sytem webhooks", "err", err)
			webhookManager.Errored(customerID, *ge.integrationInstanceID, gitlabRefType, "system", sdk.WebHookScopeSystem, err)
			return err
		}
	}

	groups, err := api.GroupsAll(ge.qc)
	if err != nil {
		return err
	}
	sdk.LogDebug(ge.logger, "groups", "groups", sdk.Stringify(groups))
	var userHasProjectWebhookAcess bool
	for _, group := range groups {
		if group.ValidTier {
			user, err := api.GroupUser(ge.qc, group, user.StrID)
			if err != nil {
				group.MarkedToCreateProjectWebHooks = true
				sdk.LogWarn(g.logger, "there was an error trying to get group user access level, will try to create project webhooks instead", "group", group.Name, "user", user.Name, "user_access_level", user.AccessLevel, "err", err)
				return err
			}
			sdk.LogDebug(ge.logger, "user", "access_level", user.AccessLevel)
			if user.AccessLevel >= api.Owner {
				userHasProjectWebhookAcess = true
				err = wr.registerWebhook(sdk.WebHookScopeOrg, group.ID, group.Name)
				if err != nil {
					group.MarkedToCreateProjectWebHooks = true
					sdk.LogWarn(g.logger, "there was an error trying to create group webhooks, will try to create project webhooks instead", "group", group.Name, "user", user.Name, "user_access_level", user.AccessLevel, "err", err)
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
				err = fmt.Errorf("error trying to get group projects err => %s", err)
				webhookManager.Errored(customerID, *integrationInstanceID, gitlabRefType, group.ID, sdk.WebHookScopeOrg, err)
				return err
			}
			for _, project := range projects {
				user, err := api.ProjectUser(ge.qc, project, user.StrID)
				if err != nil {
					err = fmt.Errorf("error trying to get project user user => %s err => %s", user.Name, err)
					webhookManager.Errored(customerID, *integrationInstanceID, gitlabRefType, project.ID, sdk.WebHookScopeProject, err)
					return err
				}
				if user.AccessLevel >= api.Owner || userHasProjectWebhookAcess {
					err = wr.registerWebhook(sdk.WebHookScopeRepo, project.ID, project.Name)
					if err != nil {
						err := fmt.Errorf("error trying to register project webhooks err => %s", err)
						webhookManager.Errored(customerID, *integrationInstanceID, gitlabRefType, project.ID, sdk.WebHookScopeProject, err)
						sdk.LogError(ge.logger, "error creating project webhook", "err", err)
						return err
					}
				} else {
					err := fmt.Errorf("at least Maintainer level access is needed to create webhooks for this project project => %s user => %s user_access_level %d", project.Name, user.Name, user.AccessLevel)
					webhookManager.Errored(customerID, *integrationInstanceID, gitlabRefType, project.ID, sdk.WebHookScopeProject, err)
					sdk.LogError(ge.logger, err.Error())
				}
			}
		}
	}

	return nil
}

type webHookRegistration struct {
	manager               sdk.WebHookManager
	customerID            string
	integrationInstanceID string
	ge                    *GitlabExport
}

func (wr *webHookRegistration) registerWebhook(whType sdk.WebHookScope, entityID, entityName string) error {
	if wr.ge.isWebHookInstalled(whType, wr.manager, wr.customerID, wr.integrationInstanceID, entityID) {
		return nil
	}

	webHooks, err := wr.ge.getHooks(whType, entityID, entityName)
	if err != nil {
		return err
	}

	var found bool
	for _, wh := range webHooks {
		if strings.Contains(wh.URL, "event.api") && strings.Contains(wh.URL, "pinpoint.com") && strings.Contains(wh.URL, wr.integrationInstanceID) {
			found = true
			break
		}
	}

	if !found {
		url, err := wr.manager.Create(wr.customerID, wr.integrationInstanceID, gitlabRefType, entityID, whType, "version="+hookVersion)
		if err != nil {
			wr.manager.Delete(wr.customerID, wr.integrationInstanceID, gitlabRefType, entityID, whType)
			return err
		}
		err = api.CreateWebHook(whType, wr.ge.qc, url, entityID, entityName)
		if err != nil {
			return err
		}
	}

	return nil
}
