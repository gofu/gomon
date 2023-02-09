package gomon

import (
	"context"
	"path"
	"runtime"
	"time"

	"golang.org/x/exp/constraints"
	"golang.org/x/sync/errgroup"
)

// fibSlice returns a slice of count elements, where every next element is
// multiplied by its fibonacci order.
//
// Example:
//
//	fibSlice[int](5, 1) // []int{1,2,3,5,8}
//	fibSlice[int](5, 2) // []int{2,4,6,10,16}
func fibSlice[T constraints.Integer | constraints.Float](count int, multiplier T) []T {
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

type envHighlighter struct {
	env EnvConfig
	Highlighter
}

// highlight stack element, ie. current file and line, and surrounding lines.
func (h *envHighlighter) highlight(root RootType, file string, hl *Highlight) error {
	file = path.Join(EnvRoot(h.env, root), file)
	return h.Highlighter.Highlight(hl, file)
}

// markupGoroutines fills highlight data for up to markupLimit goroutines.
func markupGoroutines(ctx context.Context, goroutines []Goroutine, highlighter *envHighlighter, markupLimit, wrapSize int) error {
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
				s.WrapSize = wrapSize
				err = highlighter.highlight(s.Root, s.File, &s.Highlight)
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
	return g.Wait()
}

func filterGoroutines(running []Goroutine, minDuration time.Duration) (filtered []Goroutine, skipped int) {
	if minDuration > 0 {
		for _, gr := range running {
			if gr.Duration < minDuration {
				skipped++
				continue
			}
			filtered = append(filtered, gr)
		}
	} else {
		filtered = running
	}
	return
}
