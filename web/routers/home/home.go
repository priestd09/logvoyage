package home

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/belogik/goes"
	"github.com/firstrow/logvoyage/common"
	"github.com/firstrow/logvoyage/web/context"
	"github.com/firstrow/logvoyage/web/widgets"
)

const (
	timeLayout = "2006/01/02 15:04" // Users input time format
	perPage    = 100
)

type DateTimeRange struct {
	Start string
	Stop  string
}

func (this *DateTimeRange) IsValid() bool {
	return this.Start != "" || this.Stop != ""
}

// Represents search request to perform in ES
type SearchRequest struct {
	Text      string   // test to search
	Indexes   []string // ES indexeses to perform search
	Types     []string // search types
	Size      int      // home much objects ES must return
	From      int      // how much objects should ES skip from first
	TimeRange DateTimeRange
}

func buildSearchRequest(text string, indexes []string, types []string, size int, from int, datetime DateTimeRange) SearchRequest {
	return SearchRequest{
		Text:      text,
		Indexes:   indexes,
		From:      from,
		Types:     types,
		Size:      perPage,
		TimeRange: datetime,
	}
}

// Detects time range from request and returns
// elastic compatible format string
func buildTimeRange(req *http.Request) DateTimeRange {
	var timeRange DateTimeRange

	switch req.URL.Query().Get("time") {
	case "15m":
		timeRange.Start = "now-15m"
	case "30m":
		timeRange.Start = "now-30m"
	case "60m":
		timeRange.Start = "now-60m"
	case "12h":
		timeRange.Start = "now-12h"
	case "24h":
		timeRange.Start = "now-24h"
	case "week":
		timeRange.Start = "now-1d"
	case "custom":
		timeStart, err := time.Parse(timeLayout, req.URL.Query().Get("time_start"))
		if err == nil {
			timeRange.Start = timeStart.Format(time.RFC3339)
		}
		timeStop, err := time.Parse(timeLayout, req.URL.Query().Get("time_stop"))
		if err == nil {
			timeRange.Stop = timeStop.Format(time.RFC3339)
		}
	}

	return timeRange
}

// Search logs in elastic.
func search(searchRequest SearchRequest) (goes.Response, error) {
	conn := common.GetConnection()

	if len(searchRequest.Text) > 0 {
		strconv.Quote(searchRequest.Text)
	} else {
		searchRequest.Text = "*"
	}

	var query = map[string]interface{}{
		"query": map[string]interface{}{
			"query_string": map[string]string{
				"default_field": "message",
				"query":         searchRequest.Text,
			},
		},
		"from": searchRequest.From,
		"size": searchRequest.Size,
		"sort": map[string]string{
			"datetime": "desc",
		},
	}

	if searchRequest.TimeRange.IsValid() {
		datetime := make(map[string]string)
		if searchRequest.TimeRange.Start != "" {
			datetime["gte"] = searchRequest.TimeRange.Start
		}
		if searchRequest.TimeRange.Stop != "" {
			datetime["lte"] = searchRequest.TimeRange.Stop
		}
		query["filter"] = map[string]interface{}{
			"range": map[string]interface{}{
				"datetime": datetime,
			},
		}
	}

	extraArgs := make(url.Values, 1)
	searchResults, err := conn.Search(query, searchRequest.Indexes, searchRequest.Types, extraArgs)

	if err != nil {
		return goes.Response{}, errors.New("No records found.")
	} else {
		return searchResults, nil
	}
}

func Index(ctx *context.Context) {
	query_text := ctx.Request.URL.Query().Get("q")
	types := ctx.Request.URL.Query()["types"]

	// Pagination
	pagination := widgets.NewPagination(ctx.Request)
	pagination.SetPerPage(perPage)

	println(ctx.User)

	// Load records
	searchRequest := buildSearchRequest(
		query_text,
		[]string{ctx.User.GetIndexName()},
		types,
		pagination.GetPerPage(),
		pagination.DetectFrom(),
		buildTimeRange(ctx.Request),
	)
	// Search data in elastic
	data, err := search(searchRequest)

	pagination.SetTotalRecords(data.Hits.Total)

	var viewName string
	viewData := context.ViewData{
		"logs":       data.Hits.Hits,
		"total":      data.Hits.Total,
		"took":       data.Took,
		"types":      types,
		"time":       ctx.Request.URL.Query().Get("time"),
		"time_start": ctx.Request.URL.Query().Get("time_start"),
		"time_stop":  ctx.Request.URL.Query().Get("time_stop"),
		"query_text": query_text,
		"pagination": pagination,
	}

	if err == nil {
		if ctx.Request.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			viewName = "home/table"
		} else {
			viewName = "home/index"
		}
	} else {
		viewName = "home/no_records"
	}

	ctx.HTML(viewName, viewData)
}
