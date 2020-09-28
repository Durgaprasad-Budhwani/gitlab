package api

import (
	"net/url"
	"time"

	"github.com/pinpt/agent/v4/sdk"
)

type Page func(log sdk.Logger, params url.Values, stopOnUpdatedAt time.Time) (NextPage, error)

func Paginate(log sdk.Logger, nextPage NextPage, lastProcessed time.Time, fn Page) (err error) {
	if nextPage == "" {
		nextPage = "1"
	}
	p := url.Values{}
	p.Set("per_page", "100")

	for {
		p.Set("page", string(nextPage))
		if !lastProcessed.IsZero() {
			p.Set("order_by", "updated_at")
		}
		nextPage, err = fn(log, p, lastProcessed)
		if err != nil {
			return err
		}
		if nextPage == "" {
			return nil
		}
	}
}
