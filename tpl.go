package gomon

import (
	_ "embed"
	"html/template"
	"net/url"
	"strconv"
	"time"
)

var (
	//go:embed index.gohtml
	indexTplData string
	indexTpl     = template.Must(template.New("").Funcs(template.FuncMap{
		"revIndex": func(index, length int) (revIndex int) { return (length - 1) - index },
		"sub":      func(a, b int) int { return a - b },
		"rawHTML":  func(s string) template.HTML { return template.HTML(s) },
	}).Parse(indexTplData))
)

type RequestData struct {
	// Min duration of goroutines to show.
	Min time.Duration
	// MarkupLimit is the max number of highlighted goroutines.
	MarkupLimit int
	// Lines to include before and after the current line.
	// Negative number disables highlight.
	Lines int
}

func ParseRequestData(query url.Values) (RequestData, error) {
	var data RequestData
	var errs []error
	var err error
	if min := query.Get("min"); len(min) != 0 {
		data.Min, err = time.ParseDuration(min)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if markupLimit := query.Get("markup"); len(markupLimit) != 0 {
		data.MarkupLimit, err = strconv.Atoi(markupLimit)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if linesStr := query.Get("lines"); len(linesStr) != 0 {
		data.Lines, err = strconv.Atoi(linesStr)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return data, nil
	}
	return data, errs[0] // until errors.Join
}

type IndexData struct {
	RequestData
	Durations []time.Duration
	Markups   []int
	Contexts  []int
	Total     int
	Running   []Goroutine
	Skipped   int
}
