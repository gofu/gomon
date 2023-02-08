package gomon

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"golang.org/x/sync/errgroup"
)

// StartServer starts an HTTP server on configured address, showing running
// goroutines and their call stack context, fetched from .go source files.
// Canceling ctx stops the server, and returns ctx.Err().
func StartServer(ctx context.Context, conf ServerConfig) error {
	ln, err := net.Listen("tcp", conf.Addr)
	if err != nil {
		return err
	}
	log.Printf("Listening on http://%s", ln.Addr())
	group, ctx := errgroup.WithContext(ctx)
	svc := NewHTTPProvider(conf.PProfURL, conf.Remote.WithDefaults(conf.Local))
	srv := &http.Server{
		Addr:              ln.Addr().String(),
		Handler:           NewServerHandler(conf.Local, svc),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	group.Go(func() error {
		err := srv.Serve(ln)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})
	group.Go(func() error {
		<-ctx.Done()
		_ = srv.Close()
		return ctx.Err()
	})
	return group.Wait()
}

func NewServerHandler(env EnvConfig, provider GoroutineProvider) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/", NewHandler(env, provider))
	return mux
}
