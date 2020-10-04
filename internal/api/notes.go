package api

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

// WebhookNote note struct comming from webhooks
type WebhookNote struct {
	RefID        int64  `json:"id"`
	System       bool   `json:"system"`
	Note         string `json:"note"`
	NoteType     string `json:"type"`
	URL          string `json:"url"`
	AuthorID     int64  `json:"author_id"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	NoteableType string `json:"noteable_type"`
}

// NoteDateFormat note date format
const NoteDateFormat = "2006-01-02 15:04:05 MST"

func (wn *WebhookNote) ToSourceCodePullRequestReview() (review *sdk.SourceCodePullRequestReview) {

	review.RefType = "gitlab"
	review.RefID = strconv.FormatInt(wn.RefID, 10)
	review.State = sdk.SourceCodePullRequestReviewStateCommented
	review.URL = wn.URL
	review.UserRefID = strconv.FormatInt(wn.AuthorID, 10)

	t, _ := time.Parse(NoteDateFormat, wn.CreatedAt)

	sdk.ConvertTimeToDateModel(t, &review.CreatedDate)

	return
}

// Note raw struct from api
type Note struct {
	ID     int64           `json:"id"`
	System bool            `json:"system"`
	Body   json.RawMessage `json:"body"`
	Author struct {
		ID       int64  `json:"id"`
		Username string `jsons:"username"`
	} `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

// GetGetSinglePullRequestNote get note details to write review
func GetGetSinglePullRequestNote(
	qc QueryContext,
	params url.Values,
	projectName string,
	projectRefID string,
	prRefID string,
	prIID int64,
	username string,
	prUpdatedAt string,
	action string) (pi NextPage, rnote *Note, err error) {

	sdk.LogDebug(qc.Logger, "pull request reviews", "project", projectName, "repo_ref_id", projectRefID, "pr_id", prRefID, "pr_iid", prIID, "params", params)

	objectPath := sdk.JoinURL("projects", projectRefID, "merge_requests", strconv.FormatInt(prIID, 10), "notes")

	var rnotes []*Note

	pi, err = qc.Get(objectPath, params, &rnotes)
	if err != nil {
		return
	}

	r, err := time.Parse(NoteDateFormat, prUpdatedAt)
	if err != nil {
		return
	}

	for _, note := range rnotes {
		diff1 := note.CreatedAt.Sub(r)
		diff2 := r.Sub(note.CreatedAt)
		if note.System == false &&
			note.Author.Username == username &&
			bytes.Index(note.Body, []byte(action)) == 0 &&
			((diff1 > 0 && diff1 < (1*time.Second)) ||
				(diff2 > 0 && diff2 < (1*time.Second))) {
			rnote = note
			return
		}

	}

	return
}
