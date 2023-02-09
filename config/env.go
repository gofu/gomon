package config

import (
	"fmt"
	"path"
	"strings"

	"github.com/gofu/gomon/profiler"
)

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
func (e Env) WithDefaults(defaults Env) Env {
	if len(e.Root) == 0 {
		e.Root = defaults.Root
	}
	if len(e.GoRoot) == 0 {
		e.GoRoot = defaults.GoRoot
	}
	if len(e.GoPath) == 0 {
		e.GoPath = defaults.GoPath
	}
	return e
}

// RootPath returns the corresponding .go source code root path, of type t.
func (e Env) RootPath(t profiler.RootType) string {
	switch t {
	case profiler.RootTypeProject:
		return e.Root
	case profiler.RootTypeGoRoot:
		return e.GoRoot
	case profiler.RootTypeGoPath:
		return e.GoPath
	default:
		return ""
	}
}

func (e Env) ParseFile(file string) (profiler.RootType, string, error) {
	if f, ok := cutPrefix(file, e.Root); ok {
		return profiler.RootTypeProject, f, nil
	} else if f, ok = cutPrefix(file, e.GoRoot); ok {
		return profiler.RootTypeGoRoot, f, nil
	} else if f, ok = cutPrefix(file, e.GoPath); ok {
		return profiler.RootTypeGoPath, f, nil
	} else if strings.HasPrefix(file, "_cgo_") {
		return profiler.RootTypeCGo, file, nil
	} else {
		return "", "", fmt.Errorf("unknown component path: %q", file)
	}
}

func (e Env) Normalize() Env {
	e.Root = normalizePath(e.Root)
	e.GoRoot = normalizePath(e.GoRoot)
	e.GoPath = normalizePath(e.GoPath)
	return e
}

// normalizePath replaces runs path.Clean on p, replaces backlashes
// with slashes, and appends trailing slash if p!="".
func normalizePath(p string) string {
	if len(p) == 0 {
		return ""
	}
	return strings.TrimRight(strings.ReplaceAll(path.Clean(p), "\\", "/"), "/") + "/"
}

func cutPrefix(s, prefix string) (string, bool) {
	if strings.HasPrefix(s, prefix) {
		return s[len(prefix):], true
	}
	return s, false
}
