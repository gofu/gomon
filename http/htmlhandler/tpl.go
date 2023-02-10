package htmlhandler

import (
	_ "embed"
	"html/template"
	"net/url"
	"strconv"
	"time"

	"github.com/gofu/gomon/highlight"
	"github.com/gofu/gomon/profiler"
)

var (
	//go:embed tpl.gohtml
	tplData string
	tpl     = template.Must(template.New("").Funcs(template.FuncMap{
		"revIndex": func(index, length int) (revIndex int) { return (length - 1) - index },
		"sub":      func(a, b int) int { return a - b },
		"rawHTML":  func(s string) template.HTML { return template.HTML(s) },
	}).Parse(tplData))
)

type Filter struct {
	// MinDuration duration of goroutines to show.
	MinDuration time.Duration
	// MaxDuration duration of goroutines to show.
	MaxDuration time.Duration
}

func (f Filter) IncludeAll() bool {
	return f.MinDuration == 0 && f.MaxDuration == 0
}

func (f Filter) Include(gr profiler.Goroutine) bool {
	if f.MinDuration != 0 && gr.Duration < f.MinDuration {
		return false
	}
	if f.MaxDuration != 0 && gr.Duration > f.MaxDuration {
		return false
	}
	return true
}

func (f Filter) Filter(gs []profiler.Goroutine) ([]profiler.Goroutine, int) {
	var skipped int
	if f.IncludeAll() {
		return gs, 0
	}
	var filtered []profiler.Goroutine
	for _, g := range gs {
		if !f.Include(g) {
			skipped++
			continue
		}
		filtered = append(filtered, g)
	}
	return filtered, skipped
}

type MarkupOptions struct {
	// MarkupLimit is the max number of highlighted goroutines.
	MarkupLimit int
	highlight.Options
}

type Request struct {
	Filter
	MarkupOptions
}

func ParseRequest(query url.Values) (Request, error) {
	var data Request
	var errs []error
	var err error
	if min := query.Get("min"); len(min) != 0 {
		data.MinDuration, err = time.ParseDuration(min)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if max := query.Get("max"); len(max) != 0 {
		data.MaxDuration, err = time.ParseDuration(max)
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
		data.WrapSize, err = strconv.Atoi(linesStr)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return data, nil
	}
	return data, errs[0] // until errors.Join
}

type Data struct {
	Request
	Durations []time.Duration
	Markups   []int
	Contexts  []int
	Total     int
	Running   []profiler.Goroutine
	Skipped   int
}
