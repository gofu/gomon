package htmlhandler

import (
	"context"
	"runtime"

	"github.com/gofu/gomon/highlight"
	"github.com/gofu/gomon/profiler"
	"golang.org/x/sync/errgroup"
)

// MarkupGoroutines fills highlight data (HTML) for provided goroutines.
func MarkupGoroutines(ctx context.Context, goroutines []profiler.Goroutine, highlighter highlight.Highlighter, options MarkupOptions) error {
	opts := highlight.Options{WrapSize: options.WrapSize}
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())
	var err error
	for i, gr := range goroutines {
		if options.MarkupLimit != 0 && i >= options.MarkupLimit {
			break
		}
		gr := gr
		g.Go(func() error {
			for j := range gr.CallStack {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				s := &gr.CallStack[j]
				err = highlighter.Highlight(s.FileLine, opts, &s.Highlight)
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
	return g.Wait()
}
