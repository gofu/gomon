package gomon

import (
	_ "embed"
	"html/template"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/exp/constraints"
)

var (
	//go:embed index.gohtml
	indexTplData string
	indexTpl     = template.Must(template.New("").Funcs(template.FuncMap{
		"revIndex": func(index, length int) (revIndex int) { return (length - 1) - index },
		"sub":      func(a, b int) int { return a - b },
	}).Parse(indexTplData))
)

type RequestData struct {
	Min         time.Duration
	MarkupLimit int
	Lines       int
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

func fibonacciSlice[T constraints.Integer | constraints.Float](count int, multiplier T) []T {
	elems := make([]T, count)
	for i := 0; i < count; i++ {
		switch i {
		case 0:
			elems[i] = multiplier
		case 1:
			elems[i] = 2 * multiplier
		default:
			elems[i] = elems[i-2] + elems[i-1]
		}
	}
	return elems
}
