package api

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/agent.next/sdk"
)

type WebhookNote struct {
	ID   int64  `json:"id"`
	Note string `json:"note"`
	// Author struct {
	// 	ID       int64  `json:"id"`
	// 	Username string `jsons:"username"`
	// } `json:"author"`
	// CreatedAt time.Time `json:"created_at"`
	System bool `json:"system"`
}

// Note raw struct from api
type Note struct {
	ID     int64           `json:"id"`
	Body   json.RawMessage `json:"body"`
	Author struct {
		ID       int64  `json:"id"`
		Username string `jsons:"username"`
	} `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	System    bool      `json:"system"`
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
	prUpdatedAt time.Time,
	action string) (pi NextPage, rnote *Note, err error) {

	sdk.LogDebug(qc.Logger, "pull request reviews", "project", projectName, "repo_ref_id", projectRefID, "pr_id", prRefID, "pr_iid", prIID, "params", params)

	objectPath := sdk.JoinURL("projects", projectRefID, "merge_requests", strconv.FormatInt(prIID, 10), "notes")

	var rnotes []*Note

	pi, err = qc.Get(objectPath, params, &rnotes)
	if err != nil {
		return
	}

	for _, note := range rnotes {
		diff1 := note.CreatedAt.Sub(prUpdatedAt)
		diff2 := prUpdatedAt.Sub(note.CreatedAt)
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
