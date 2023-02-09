package gomon

import (
	"context"
	_ "embed"
	"html/template"
	"net/url"
	"path"
	"runtime"
	"strconv"
	"time"

	"golang.org/x/exp/constraints"
	"golang.org/x/sync/errgroup"
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

type EnvHighlighter struct {
	Env EnvConfig
	Highlighter
}

func (h *EnvHighlighter) Highlight(elem *StackElem) error {
	return h.Highlighter.Highlight(&elem.Highlight, path.Join(elem.Root.FromEnv(h.Env), elem.File))
}

func markupGoroutines(ctx context.Context, goroutines []Goroutine, highlighter *EnvHighlighter, markupLimit, wrapSize int) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())
	var err error
	for i, goroutine := range goroutines {
		if markupLimit != 0 && i > markupLimit {
			break
		}
		goroutine := goroutine
		g.Go(func() error {
			for j := range goroutine.Stack {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				s := &goroutine.Stack[j]
				s.HighlightLine = s.Line
				s.WrapSize = wrapSize
				err = highlighter.Highlight(s)
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
	return g.Wait()
}
