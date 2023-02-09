package gomon

import (
	"context"
	"runtime"

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

// markupGoroutines fills highlight data for up to markupLimit goroutines.
func markupGoroutines(ctx context.Context, goroutines []Goroutine, highlighter Highlighter, options MarkupOptions) error {
	opts := HighlightOptions{WrapSize: options.WrapSize}
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())
	var err error
	for i, gr := range goroutines {
		if options.MarkupLimit != 0 && i >= options.MarkupLimit {
			break
		}
		gr := gr
		g.Go(func() error {
			for j := range gr.Stack {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				s := &gr.Stack[j]
				err = highlighter.Highlight(s.FileInfo, opts, &s.Highlight)
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
	return g.Wait()
}
