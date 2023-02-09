// Package config contains configuration structs.
package config

import "github.com/gofu/gomon/profiler"

// Server configuration, for running an HTTP server serving detailed profiler info.
type Server struct {
	// Addr is the HTTP address to listen on.
	Addr string
	// PProfURL is the remote /debug/pprof URL to query.
	PProfURL string
	// Local environment info, used to parse .go source files.
	Local Env
	// Remote environment info, used to map results of PProfURL to Local environment.
	Remote Env
}

// Env contains absolute paths that can contain .go source files.
// on a single environment.
type Env struct {
	// Root of the project source code root path.
	Root string
	// GoRoot is the GOROOT environment variable.
	GoRoot string
	// GoPath is the GOPATH environment variable.
	GoPath string
}

// WithDefaults returns a new Env, with empty string
// values defaulting to values from conf.
func (c Env) WithDefaults(defaults Env) Env {
	if len(c.Root) == 0 {
		c.Root = defaults.Root
	}
	if len(c.GoRoot) == 0 {
		c.GoRoot = defaults.GoRoot
	}
	if len(c.GoPath) == 0 {
		c.GoPath = defaults.GoPath
	}
	return c
}

// RootPath returns the corresponding .go source code root path, of type t.
func (c Env) RootPath(t profiler.RootType) string {
	switch t {
	case profiler.RootTypeProject:
		return c.Root
	case profiler.RootTypeGoRoot:
		return c.GoRoot
	case profiler.RootTypeGoPath:
		return c.GoPath
	default:
		return ""
	}
}
