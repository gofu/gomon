// Command gomon starts a local HTTP server showing running goroutines from a remote Go
// process's /debug/pprof HTTP page, using local source .go files to show stack trace.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"

	"github.com/gofu/gomon/server"
)

func main() {
	var s server.Server
	flag.StringVar(&s.Addr, "addr", "127.0.0.1:7656", "HTTP listen address")
	flag.StringVar(&s.PProfURL, "url", "http://127.0.0.1:7656/debug/pprof", "Remote /debug/pprof URL")
	flag.StringVar(&s.Local.Root, "local-root", currentDir(), "Local project root")
	flag.StringVar(&s.Local.GoRoot, "local-goroot", runtime.GOROOT(), "Local GOROOT")
	flag.StringVar(&s.Local.GoPath, "local-gopath", os.Getenv("GOPATH"), "Local GOPATH")
	flag.StringVar(&s.Remote.Root, "remote-root", "", "Remote project root")
	flag.StringVar(&s.Remote.GoRoot, "remote-goroot", "", "Remote GOROOT")
	flag.StringVar(&s.Remote.GoPath, "remote-gopath", "", "Remote GOPATH")
	flag.Var((*sliceFlags)(&s.RemoteSSH), "remote-ssh", "Remote SSH connections to reach process host machine, for direct memory reading")
	flag.Parse()
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	err := server.ListenAndServe(ctx, s)
	if err != nil {
		log.Fatal(err)
	}
}

func currentDir() string {
	wd, _ := os.Getwd()
	return wd
}

type sliceFlags []string

func (i *sliceFlags) String() string {
	return strings.Join(*i, " ")
}

func (i *sliceFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}
