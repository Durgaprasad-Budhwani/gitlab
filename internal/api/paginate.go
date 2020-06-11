package api

import (
	"errors"
	"net/url"
	"time"

	"github.com/pinpt/agent.next/sdk"
)

type PaginateStartAtFn func(log sdk.Logger, paginationParams url.Values) (page PageInfo, _ error)

func PaginateStartAt(log sdk.Logger, nextPage string, fn PaginateStartAtFn) error {
	if nextPage == "" {
		nextPage = "1"
	}
	for {
		q := url.Values{}
		q.Add("page", nextPage)
		pageInfo, err := fn(log, q)
		if err != nil {
			return err
		}
		if pageInfo.NextPage == "" {
			return nil
		}
		if pageInfo.PageSize == 0 {
			return errors.New("pageSize is 0")
		}

		nextPage = pageInfo.NextPage
	}
}

type PaginateNewerThanFn func(log sdk.Logger, parameters url.Values, stopOnUpdatedAt time.Time) (PageInfo, error)

func PaginateNewerThan(log sdk.Logger, lastProcessed time.Time, fn PaginateNewerThanFn) error {
	nextPage := "1"
	p := url.Values{}
	p.Set("per_page", "100")

	for {
		p.Add("page", nextPage)
		if !lastProcessed.IsZero() {
			p.Add("order_by", "updated_at")
		}
		pageInfo, err := fn(log, p, lastProcessed)
		if err != nil {
			return err
		}
		if pageInfo.NextPage == "" {
			return nil
		}
		if pageInfo.PageSize == 0 {
			return errors.New("pageSize is 0")
		}
		nextPage = pageInfo.NextPage
	}
}
