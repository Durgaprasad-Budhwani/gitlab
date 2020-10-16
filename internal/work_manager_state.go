package internal

import (
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

const workManagerKey = "work_manager"

type recover struct {
	RefProject map[string]map[string]*issueDetail
	RefIssues  map[string]string
}

// Persist persist info
func (w *WorkManager) Persist() error {

	refProjectMap := make(map[string]map[string]*issueDetail, 0)
	w.refProject.Range(func(k, v interface{}) bool {
		refProjectMap[k.(string)] = v.(map[string]*issueDetail)
		return true
	})

	refIssuesMap := make(map[string]string, 0)
	w.refIssueDetails.Range(func(k, v interface{}) bool {
		refIssuesMap[k.(string)] = v.(string)
		return true
	})

	r := recover{
		RefProject: refProjectMap,
		RefIssues:  refIssuesMap,
	}

	start := time.Now()
	err := w.state.Set(workManagerKey, r)
	sdk.LogDebug(w.logger, "persistence took", "time", time.Since(start))

	return err
}

// Restore restore info into work manager
func (w *WorkManager) Restore() error {

	var r recover

	start := time.Now()
	ok, err := w.state.Get(workManagerKey, &r)
	sdk.LogDebug(w.logger, "recovery took", "time", time.Since(start))
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	for k, v := range r.RefProject {
		w.refProject.Store(k, v)
	}

	for k, v := range r.RefIssues {
		w.refIssueDetails.Store(k, v)
	}

	return nil
}

// Delete delete state
func (w *WorkManager) Delete() error {
	return w.state.Delete(workManagerKey)
}
