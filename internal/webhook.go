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

const hookVersion = "2" // change this to upgrade the hook in case the events change

type user struct {
	Name      string `json:"name"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

func (u *user) RefID(customerID string) string {
	return sdk.Hash(customerID, u.Email)
}

func (a *user) ToModel(customerID string, integrationInstanceID string) *sdk.SourceCodeUser {
	user := &sdk.SourceCodeUser{}
	user.CustomerID = customerID
	user.RefID = a.RefID(customerID)
	user.RefType = "gitlab"
	user.IntegrationInstanceID = sdk.StringPointer(integrationInstanceID)
	user.URL = sdk.StringPointer("")
	user.AvatarURL = sdk.StringPointer(a.AvatarURL)
	user.Email = sdk.StringPointer(a.Email)
	user.Name = a.Name
	var userType sdk.SourceCodeUserType
	if strings.Contains(a.Name, "Bot") {
		userType = sdk.SourceCodeUserTypeBot
	} else {
		userType = sdk.SourceCodeUserTypeHuman
	}

	user.Type = userType
	user.Username = sdk.StringPointer(a.Username)

	return user
}

type webHookRootPayload struct {
	WebHookMainObject json.RawMessage `json:"object_attributes"`
	Project           struct {
		Name string `json:"name"`
		ID   int64  `json:"id"`
	} `json:"project"`
	User         user                   `json:"user"`
	Changes      json.RawMessage        `json:"changes"`
	MergeRequest api.WebhookPullRequest `json:"merge_request"`
	EventName    string                 `json:"event_name"`
	ProjectID    int64                  `json:"project_id"`
	UserID       int64                  `json:"user_id"`
	Assignees    []user                 `json:"assignees"`
}

// WebHook is called when a webhook is received on behalf of the integration
func (i *GitlabIntegration) WebHook(webhook sdk.WebHook) (rerr error) {

	customerID := webhook.CustomerID()
	integrationInstanceID := webhook.IntegrationInstanceID()

	logger := sdk.LogWith(i.logger, "entity", "webhook", "customer_id", customerID)

	pipe := webhook.Pipe()

	event := webhook.Headers()["x-gitlab-event"]

	state := webhook.State()

	userManager := NewUserManager(customerID, webhook, state, pipe, integrationInstanceID)

	ge, err := i.SetQueryConfig(logger, webhook.Config(), i.manager, customerID)
	if err != nil {
		rerr = err
		return
	}

	ge.qc.Pipe = pipe
	ge.qc.UserManager = userManager
	ge.qc.WorkManager = NewWorkManager(logger, state)

	sdk.LogInfo(logger, "event", "event", event)

	sdk.LogDebug(logger, "webhook-body", "body", string(webhook.Bytes()))

	var rootWebHookObject webHookRootPayload
	rerr = json.Unmarshal(webhook.Bytes(), &rootWebHookObject)
	if rerr != nil {
		sdk.LogError(logger, "err", rerr)
		return
	}

	rerr = userManager.EmitGitUser(logger, &rootWebHookObject.User)
	if rerr != nil {
		sdk.LogError(logger, "err", rerr)
		return
	}

	for _, user := range rootWebHookObject.Assignees {
		rerr = userManager.EmitGitUser(logger, &user)
		if rerr != nil {
			sdk.LogError(logger, "err", rerr)
			return
		}
	}

	projectRefID := strconv.FormatInt(rootWebHookObject.Project.ID, 10)

	repoID := sdk.NewSourceCodeRepoID(customerID, projectRefID, gitlabRefType)
	projectID := sdk.NewWorkProjectID(customerID, projectRefID, gitlabRefType)

	switch event {
	case "Issue Hook":
		sdk.LogInfo(logger, "recovering work manager state")
		if err := ge.qc.WorkManager.Restore(); err != nil {
			sdk.LogError(logger, "error recovering work manager state", "err", err)
			return err
		}
		issue := &api.IssueWebHook{}
		err := json.Unmarshal(rootWebHookObject.WebHookMainObject, issue)
		if err != nil {
			return err
		}
		// Fetch Issue
		project := &sdk.SourceCodeRepo{}
		project.ID = projectID
		project.Name = rootWebHookObject.Project.Name
		project.RefID = projectRefID
		if err := ge.writeSingleIssue(project, issue.IID); err != nil {
			return err
		}
		sdk.LogInfo(logger, "persisting work manager into state")
		if err := ge.qc.WorkManager.Persist(); err != nil {
			sdk.LogError(logger, "error persisting work manager state", "err", err)
			return err
		}
	case "Merge Request Hook":

		pr := &api.WebhookPullRequest{}
		rerr = json.Unmarshal(rootWebHookObject.WebHookMainObject, pr)
		if rerr != nil {
			return
		}

		var scPr = &sdk.SourceCodePullRequest{}
		scPr, rerr = pr.ToSourceCodePullRequest(logger, customerID, repoID, gitlabRefType)
		if rerr != nil {
			return
		}
		scPr.IntegrationInstanceID = &integrationInstanceID

		switch scPr.Status {
		case sdk.SourceCodePullRequestStatusClosed:
			scPr.ClosedByRefID = rootWebHookObject.User.RefID(customerID)
		case sdk.SourceCodePullRequestStatusMerged:
			scPr.MergedByRefID = rootWebHookObject.User.RefID(customerID)
		}

		repo := &sdk.SourceCodeRepo{}
		repo.Name = rootWebHookObject.Project.Name
		repo.RefID = projectRefID

		prr := api.PullRequest{}
		prr.SourceCodePullRequest = scPr
		prr.IID = strconv.FormatInt(pr.IID, 10)
		_, _, rerr = api.PullRequestReviews(ge.qc, repo, prr, nil)
		if rerr != nil {
			return
		}

		sdk.LogDebug(logger, "source code pull request", "body", scPr.Stringify())

		rerr = pipe.Write(scPr)
		if rerr != nil {
			return
		}

		pullRequestID := sdk.NewSourceCodePullRequestID(customerID, scPr.RefID, gitlabRefType, repoID)

		switch pr.Action {
		case "approved", "unapproved":
			review, err := i.GetReviewFromAction(
				logger,
				ge.qc,
				customerID,
				rootWebHookObject.Project.Name,
				repoID,
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

			var changes map[string]json.RawMessage

			rerr = json.Unmarshal(rootWebHookObject.Changes, &changes)
			if rerr != nil {
				return
			}

			_, updateKeyExist = changes["updated_at"]
			keyCount++
			if keyCount > 1 {
				return
			}
			if !updateKeyExist {
				return
			}
			var updateat time.Time
			updateat, rerr = time.Parse("2006-01-02 15:04:05 MST", pr.UpdatedAt)
			if rerr != nil {
				return
			}

			repo := &sdk.SourceCodeRepo{}
			repo.Name = rootWebHookObject.Project.Name
			repo.RefID = projectRefID

			var pr2 = &api.PullRequest{SourceCodePullRequest: &sdk.SourceCodePullRequest{}}
			pr2.IID = strconv.FormatInt(pr.IID, 10)
			pr2.RefID = scPr.RefID

			commits, err := ge.FetchPullRequestsCommitsAfter(repo, *pr2, updateat)
			if err != nil {
				return fmt.Errorf("error fetching pull requests commits on webhook, err %d", err)
			}

			sdk.LogDebug(logger, "commits found", "len", len(commits))

			for _, c := range commits {
				c.IntegrationInstanceID = &integrationInstanceID
				rerr = pipe.Write(c)
				if rerr != nil {
					return
				}
			}
		case "open", "reopen":

			repo := &sdk.SourceCodeRepo{}
			repo.Name = rootWebHookObject.Project.Name
			repo.RefID = projectRefID
			var pr2 = &api.PullRequest{SourceCodePullRequest: &sdk.SourceCodePullRequest{}}
			pr2.IID = strconv.FormatInt(pr.IID, 10)
			pr2.RefID = scPr.RefID

			commits, err := ge.fetchPullRequestsCommits(repo, *pr2)
			if err != nil {
				return fmt.Errorf("error fetching pull requests commits on webhook, err %d", err)
			}

			sdk.LogDebug(logger, "commits found", "len", len(commits))

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
			var scPr = &sdk.SourceCodePullRequest{}
			scPr, rerr = rootWebHookObject.MergeRequest.ToSourceCodePullRequest(logger, customerID, repoID, gitlabRefType)
			if rerr != nil {
				return
			}

			if note.NoteType == "DiffNote" {
				review := note.ToSourceCodePullRequestReview()
				review.CustomerID = customerID
				review.IntegrationInstanceID = sdk.StringPointer(integrationInstanceID)
				review.PullRequestID = scPr.ID
				review.RepoID = repoID

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

				prComment.RepoID = repoID
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
			user, err := api.UserByID(ge.qc, rootWebHookObject.UserID)
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
	prUpdatedAt string,
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
